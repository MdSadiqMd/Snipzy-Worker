#!/usr/bin/env bash
# First Started using Private ECR but it costs a lot
# Next moved to Public ECR, but it only works when both lambda and ecr are in same region, and public ECR are for namesake global but they only work in us-east-1, us-central-1
# Next moved to Docker Hub, but AWS needs vendor lock in and it won't allow us to use external container image services
# Next moved to S3, packaging all the code with ffmpeg and ytdlp binaries and uplaoding to S3 and use it when needed, still costs, but very cheap
set -euo pipefail

# ========== Configuration ==========
export AWS_REGION="ap-south-1"
export AWS_ACCOUNT_ID=""
export LAMBDA_ROLE_ARN="arn:aws:iam::${AWS_ACCOUNT_ID}:role/Snipzy"
export REPO_NAME="snipzy-worker"
export TAG="latest"
export FUNCTION_NAME="${REPO_NAME}-${TAG}"
export S3_BUCKET="${REPO_NAME}-deploy-bucket"
export S3_KEY="package.zip"
# ====================================

echo "→ Preparing layers…"
mkdir -p layers/ffmpeg-layer/bin layers/ytdlp-layer/bin

if [[ ! -f layers/ffmpeg-layer/bin/ffmpeg ]]; then
  echo "  • Downloading and extracting ffmpeg+ffprobe…"
  curl -sL https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz -o /tmp/ffmpeg.tar.xz
  tar -xJf /tmp/ffmpeg.tar.xz --wildcards --strip-components=1 \
    -C layers/ffmpeg-layer/bin "ffmpeg-*-amd64-static/ffmpeg" "ffmpeg-*-amd64-static/ffprobe"
  chmod +x layers/ffmpeg-layer/bin/ffmpeg layers/ffmpeg-layer/bin/ffprobe
  rm /tmp/ffmpeg.tar.xz
  echo "  ✓ FFmpeg and ffprobe downloaded and extracted"
fi

if [[ ! -f layers/ytdlp-layer/bin/yt-dlp ]]; then
  echo "  • Downloading yt-dlp…"
  curl -sL https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp \
    -o layers/ytdlp-layer/bin/yt-dlp
  chmod +x layers/ytdlp-layer/bin/yt-dlp
  echo "  ✓ yt-dlp downloaded"
fi

echo "→ Verifying binaries…"
ls -lh layers/ffmpeg-layer/bin/
ls -lh layers/ytdlp-layer/bin/

echo "→ Building Go binary…"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags lambda.norpc -o bootstrap ./cmd/main.go

echo "→ Packaging deployment ZIP…"
rm -f package.zip

mkdir -p bin
cp bootstrap bin/
cp layers/ffmpeg-layer/bin/ffmpeg bin/
cp layers/ffmpeg-layer/bin/ffprobe bin/
cp layers/ytdlp-layer/bin/yt-dlp bin/
chmod +x bin/*
zip -r package.zip bin/
rm -rf bin/

echo "→ Ensuring S3 bucket exists…"
aws s3api create-bucket --bucket "$S3_BUCKET" --region "$AWS_REGION" --create-bucket-configuration LocationConstraint="$AWS_REGION" 2>/dev/null || echo "  ℹ Bucket already exists"

echo "→ Uploading package to S3…"
aws s3 cp package.zip s3://"$S3_BUCKET"/"$S3_KEY" --region "$AWS_REGION"

echo "→ Checking if Lambda function exists…"
FUNCTION_EXISTS=false
CURRENT_PACKAGE_TYPE=""

if aws lambda get-function --function-name "$FUNCTION_NAME" --region "$AWS_REGION" &>/dev/null; then
  FUNCTION_EXISTS=true
  CURRENT_PACKAGE_TYPE=$(aws lambda get-function --function-name "$FUNCTION_NAME" --region "$AWS_REGION" --query 'Configuration.PackageType' --output text)
  echo "  ✓ Function exists with package type: $CURRENT_PACKAGE_TYPE"
  
  if [[ "$CURRENT_PACKAGE_TYPE" == "Image" ]]; then
    echo "  • Function is currently Image-based, recreating as Zip-based…"
    aws lambda delete-function --function-name "$FUNCTION_NAME" --region "$AWS_REGION"
    echo "  ✓ Existing Image function deleted"
    sleep 5
    FUNCTION_EXISTS=false
  fi
fi

if [[ "$FUNCTION_EXISTS" == "true" && "$CURRENT_PACKAGE_TYPE" == "Zip" ]]; then
  echo "  ✓ Updating existing Zip-based function…"
  
  aws lambda update-function-code \
    --function-name "$FUNCTION_NAME" \
    --s3-bucket "$S3_BUCKET" \
    --s3-key "$S3_KEY" \
    --region "$AWS_REGION"
  
  echo "  • Waiting for function update to complete…"
  aws lambda wait function-updated \
    --function-name "$FUNCTION_NAME" \
    --region "$AWS_REGION"
  
  echo "  • Updating environment variables…"
  aws lambda update-function-configuration \
    --function-name "$FUNCTION_NAME" \
    --environment Variables="{FFMPEG_PATH=/var/task/bin/ffmpeg,YTDLP_PATH=/var/task/bin/yt-dlp,MAX_CLIP_DURATION=30,ENVIRONMENT=dev}" \
    --region "$AWS_REGION"
    
  echo "  • Waiting for configuration update to complete…"
  aws lambda wait function-updated \
    --function-name "$FUNCTION_NAME" \
    --region "$AWS_REGION"
  
  echo "  ✓ Function code and configuration updated successfully"

else
  echo "  • Creating new Zip-based Lambda function…"
  
  aws lambda create-function \
    --function-name "$FUNCTION_NAME" \
    --runtime provided.al2023 \
    --handler bootstrap \
    --package-type Zip \
    --code S3Bucket="$S3_BUCKET",S3Key="$S3_KEY" \
    --role "$LAMBDA_ROLE_ARN" \
    --timeout 900 \
    --memory-size 2048 \
    --environment Variables="{FFMPEG_PATH=/var/task/bin/ffmpeg,YTDLP_PATH=/var/task/bin/yt-dlp,MAX_CLIP_DURATION=30,ENVIRONMENT=dev}" \
    --ephemeral-storage '{"Size":10240}' \
    --architectures x86_64 \
    --region "$AWS_REGION"
  
  echo "  • Waiting for function to be active…"
  aws lambda wait function-active \
    --function-name "$FUNCTION_NAME" \
    --region "$AWS_REGION"
  
  echo "  ✓ Function created successfully"
fi

echo "→ Checking Function URL configuration…"
FUNCTION_URL=""
if aws lambda get-function-url-config --function-name "$FUNCTION_NAME" --region "$AWS_REGION" &>/dev/null; then
  echo "  ✓ Function URL already exists"
  FUNCTION_URL=$(aws lambda get-function-url-config \
    --function-name "$FUNCTION_NAME" \
    --region "$AWS_REGION" \
    --query FunctionUrl --output text)
else
  echo "  • Creating Function URL with streaming support and CORS…"
  aws lambda create-function-url-config \
    --function-name "$FUNCTION_NAME" \
    --auth-type NONE \
    --invoke-mode RESPONSE_STREAM \
    --cors '{"AllowOrigins":["*"],"AllowMethods":["POST","OPTIONS","GET"],"AllowHeaders":["Content-Type","Authorization"],"MaxAge":86400}' \
    --region "$AWS_REGION" > /tmp/function-url-config.json

  FUNCTION_URL=$(grep -o '"FunctionUrl":"[^"]*"' /tmp/function-url-config.json | cut -d'"' -f4)
  echo "  ✓ Function URL created successfully with CORS"
  rm -f /tmp/function-url-config.json
fi

echo ""
echo "✅ Deployment completed successfully!"
echo ""

echo "📋 Function Details:"
aws lambda get-function \
  --function-name "$FUNCTION_NAME" \
  --region "$AWS_REGION" \
  --query 'Configuration.[FunctionName,State,LastModified,MemorySize,Timeout,PackageType]' \
  --output table

echo ""
echo "🔗 Function URL:"
if [[ -n "$FUNCTION_URL" ]]; then
  echo "$FUNCTION_URL"
else
  echo "Not configured - create manually using AWS Console or:"
  echo "aws lambda create-function-url-config --function-name $FUNCTION_NAME --auth-type NONE --invoke-mode RESPONSE_STREAM --region $AWS_REGION"
fi

if [[ -n "$FUNCTION_URL" ]]; then
  echo ""
  echo "🧪 Test your deployment:"
  echo "curl -X POST -H 'Content-Type: application/json' \\"
  echo "  -d '{\"url\":\"https://www.youtube.com/watch?v=dQw4w9WgXcQ\",\"start\":10,\"end\":20,\"platform\":\"youtube\",\"quality\":\"medium\"}' \\"
  echo "  '$FUNCTION_URL' \\"
  echo "  --output clip.mp4"
  
  echo ""
  echo "📊 Monitor logs:"
  echo "aws logs tail /aws/lambda/$FUNCTION_NAME --follow --region $AWS_REGION"
  
  echo ""
  echo "🔍 Quick test (just get response headers):"
  echo "curl -I -X POST -H 'Content-Type: application/json' \\"
  echo "  -d '{\"url\":\"https://www.youtube.com/watch?v=test\",\"start\":0,\"end\":5,\"platform\":\"youtube\",\"quality\":\"medium\"}' \\"
  echo "  '$FUNCTION_URL'"
  
  echo ""
  echo "🗑️  Delete function (if needed):"
  echo "aws lambda delete-function --function-name $FUNCTION_NAME --region $AWS_REGION"
fi

echo "✨ Done!"
