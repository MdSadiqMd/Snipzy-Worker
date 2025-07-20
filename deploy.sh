#!/usr/bin/env bash

set -euo pipefail

# ========== Configuration ==========
export AWS_REGION=""
export AWS_ACCOUNT_ID=""
export LAMBDA_ROLE_ARN=""
export REPO_NAME="snipzy-worker"
export TAG="latest"
export FUNCTION_NAME="${REPO_NAME}-${TAG}"
# ====================================

# Use private ECR format
IMAGE_URI="$AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/$REPO_NAME:$TAG"

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

echo "→ Building Docker image…"
docker build -t $REPO_NAME:$TAG .

echo "→ Tagging and pushing to Private ECR…"
echo "  • Authenticating to private ECR..."

# Authenticate to private ECR
export DOCKER_CONFIG=$(mktemp -d)
aws ecr get-login-password --region $AWS_REGION \
  | docker login --username AWS --password-stdin $AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com

# Create private repository if it doesn't exist
aws ecr create-repository \
  --repository-name $REPO_NAME \
  --region $AWS_REGION \
  --image-scanning-configuration scanOnPush=true \
  2>/dev/null || echo "  ℹ Private repository already exists"

docker tag $REPO_NAME:$TAG $IMAGE_URI
echo "  • Pushing $IMAGE_URI…"
docker push $IMAGE_URI

# Cleanup temporary config
rm -rf "$DOCKER_CONFIG"
unset DOCKER_CONFIG

echo "→ Checking if Lambda function exists…"
FUNCTION_EXISTS=false
if aws lambda get-function --function-name $FUNCTION_NAME --region $AWS_REGION &>/dev/null; then
  FUNCTION_EXISTS=true
  echo "  ✓ Function exists, updating code…"
  
  aws lambda update-function-code \
    --function-name $FUNCTION_NAME \
    --image-uri $IMAGE_URI \
    --region $AWS_REGION
  
  echo "  • Waiting for function update to complete…"
  aws lambda wait function-updated \
    --function-name $FUNCTION_NAME \
    --region $AWS_REGION
  
  echo "  ✓ Function code updated successfully"
  
else
  echo "  • Function doesn't exist, creating new function…"
  
  aws lambda create-function \
    --function-name $FUNCTION_NAME \
    --package-type Image \
    --code ImageUri=$IMAGE_URI \
    --role $LAMBDA_ROLE_ARN \
    --timeout 900 \
    --memory-size 2048 \
    --environment Variables="{FFMPEG_PATH=/opt/bin/ffmpeg,YTDLP_PATH=/opt/bin/yt-dlp,MAX_CLIP_DURATION=30,ENVIRONMENT=dev}" \
    --ephemeral-storage '{"Size":10240}' \
    --architectures x86_64 \
    --region $AWS_REGION

  echo "  • Waiting for function to be active…"
  aws lambda wait function-active \
    --function-name $FUNCTION_NAME \
    --region $AWS_REGION
  
  echo "  ✓ Function created successfully"
fi

echo "→ Checking Function URL configuration…"
FUNCTION_URL=""
if aws lambda get-function-url-config --function-name $FUNCTION_NAME --region $AWS_REGION &>/dev/null; then
  echo "  ✓ Function URL already exists"
  FUNCTION_URL=$(aws lambda get-function-url-config \
    --function-name $FUNCTION_NAME \
    --region $AWS_REGION \
    --query FunctionUrl --output text)
else
  echo "  • Creating Function URL with streaming support…"
  
  # Create function URL with streaming support
  if aws lambda create-function-url-config \
    --function-name $FUNCTION_NAME \
    --auth-type NONE \
    --invoke-mode RESPONSE_STREAM \
    --region $AWS_REGION > /tmp/function-url-config.json 2>&1; then
    
    FUNCTION_URL=$(grep -o '"FunctionUrl":"[^"]*"' /tmp/function-url-config.json | cut -d'"' -f4)
    echo "  ✓ Function URL created successfully"
  else
    echo "  ❌ Failed to create Function URL:"
    cat /tmp/function-url-config.json
    FUNCTION_URL=""
  fi
  
  # Cleanup
  rm -f /tmp/function-url-config.json
fi

echo ""
echo "✅ Deployment completed successfully!"
echo ""

# Show function details
echo "📋 Function Details:"
aws lambda get-function \
  --function-name $FUNCTION_NAME \
  --region $AWS_REGION \
  --query 'Configuration.[FunctionName,State,LastModified,MemorySize,Timeout]' \
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
