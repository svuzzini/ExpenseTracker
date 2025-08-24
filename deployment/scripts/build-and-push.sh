#!/bin/bash

# Build and Push Script for ExpenseTracker
# Automates Docker image building and pushing to container registries

set -e

# Default values
REGISTRY=""
IMAGE_NAME="expense-tracker"
TAG="latest"
PLATFORM="linux/amd64"
BUILD_ARGS=""
PUSH=false
DOCKERFILE="deployment/docker/Dockerfile"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to show usage
show_usage() {
    echo "ExpenseTracker Build and Push Script"
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -r, --registry REGISTRY    Container registry URL (required for push)"
    echo "  -i, --image IMAGE          Image name [default: expense-tracker]"
    echo "  -t, --tag TAG              Image tag [default: latest]"
    echo "  -p, --platform PLATFORM   Target platform [default: linux/amd64]"
    echo "  -f, --dockerfile FILE      Dockerfile path [default: deployment/docker/Dockerfile]"
    echo "  -b, --build-arg ARG        Build argument (can be used multiple times)"
    echo "  --push                     Push image to registry after building"
    echo "  --multi-platform           Build for multiple platforms (linux/amd64,linux/arm64)"
    echo "  -h, --help                 Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                                           # Build local image"
    echo "  $0 --push -r docker.io/username             # Build and push to Docker Hub"
    echo "  $0 --push -r 123456789.dkr.ecr.us-west-2.amazonaws.com  # Push to AWS ECR"
    echo "  $0 -t v1.2.3 --push -r gcr.io/project-id    # Build specific version and push to GCR"
    echo "  $0 --multi-platform --push -r registry.com/user  # Multi-platform build"
    echo ""
    echo "Registry Examples:"
    echo "  Docker Hub:        docker.io/username"
    echo "  AWS ECR:           123456789.dkr.ecr.region.amazonaws.com"
    echo "  Google GCR:        gcr.io/project-id"
    echo "  GitHub Registry:   ghcr.io/username"
    echo "  Private Registry:  registry.company.com"
}

# Function to check prerequisites
check_prerequisites() {
    print_info "Checking prerequisites..."
    
    if ! command -v docker >/dev/null 2>&1; then
        print_error "Docker is not installed or not in PATH"
        exit 1
    fi
    
    # Check if Docker daemon is running
    if ! docker info >/dev/null 2>&1; then
        print_error "Docker daemon is not running"
        exit 1
    fi
    
    # Check if dockerfile exists
    if [[ ! -f "$DOCKERFILE" ]]; then
        print_error "Dockerfile not found: $DOCKERFILE"
        exit 1
    fi
    
    print_success "Prerequisites check passed"
}

# Function to detect registry type and configure accordingly
detect_registry() {
    if [[ -z "$REGISTRY" ]]; then
        print_info "No registry specified, building local image only"
        return
    fi
    
    print_info "Detected registry: $REGISTRY"
    
    # AWS ECR
    if [[ "$REGISTRY" == *"ecr"* ]]; then
        print_info "Configuring for AWS ECR..."
        configure_aws_ecr
    # Google Container Registry
    elif [[ "$REGISTRY" == "gcr.io"* ]]; then
        print_info "Configuring for Google Container Registry..."
        configure_gcr
    # GitHub Container Registry
    elif [[ "$REGISTRY" == "ghcr.io"* ]]; then
        print_info "Configuring for GitHub Container Registry..."
        configure_ghcr
    # Docker Hub
    elif [[ "$REGISTRY" == "docker.io"* ]] || [[ "$REGISTRY" != *"."* ]]; then
        print_info "Configuring for Docker Hub..."
        configure_docker_hub
    else
        print_info "Using generic registry configuration..."
    fi
}

# Function to configure AWS ECR
configure_aws_ecr() {
    if ! command -v aws >/dev/null 2>&1; then
        print_warning "AWS CLI not found. Please install and configure AWS CLI"
        print_info "Or manually run: aws ecr get-login-password --region REGION | docker login --username AWS --password-stdin $REGISTRY"
        return
    fi
    
    # Extract region from ECR URL
    REGION=$(echo "$REGISTRY" | cut -d'.' -f4)
    
    print_info "Logging into AWS ECR (region: $REGION)..."
    aws ecr get-login-password --region "$REGION" | docker login --username AWS --password-stdin "$REGISTRY"
    
    # Create repository if it doesn't exist
    REPO_NAME=$(echo "$REGISTRY/$IMAGE_NAME" | cut -d'/' -f2-)
    if ! aws ecr describe-repositories --repository-names "$REPO_NAME" --region "$REGION" >/dev/null 2>&1; then
        print_info "Creating ECR repository: $REPO_NAME"
        aws ecr create-repository --repository-name "$REPO_NAME" --region "$REGION"
    fi
}

# Function to configure Google Container Registry
configure_gcr() {
    if ! command -v gcloud >/dev/null 2>&1; then
        print_warning "Google Cloud SDK not found. Please install and configure gcloud"
        print_info "Or manually run: gcloud auth configure-docker"
        return
    fi
    
    print_info "Configuring Docker for Google Container Registry..."
    gcloud auth configure-docker --quiet
}

# Function to configure GitHub Container Registry
configure_ghcr() {
    if [[ -z "$GITHUB_TOKEN" ]]; then
        print_warning "GITHUB_TOKEN environment variable not set"
        print_info "Please set GITHUB_TOKEN or manually login: echo \$GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin"
        return
    fi
    
    print_info "Logging into GitHub Container Registry..."
    echo "$GITHUB_TOKEN" | docker login ghcr.io -u "$GITHUB_USER" --password-stdin
}

# Function to configure Docker Hub
configure_docker_hub() {
    if [[ -z "$DOCKER_USERNAME" ]] || [[ -z "$DOCKER_PASSWORD" ]]; then
        print_warning "DOCKER_USERNAME and DOCKER_PASSWORD environment variables not set"
        print_info "Please set credentials or manually login: docker login"
        return
    fi
    
    print_info "Logging into Docker Hub..."
    echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin
}

# Function to build Docker image
build_image() {
    local full_image_name
    
    if [[ -n "$REGISTRY" ]]; then
        full_image_name="$REGISTRY/$IMAGE_NAME:$TAG"
    else
        full_image_name="$IMAGE_NAME:$TAG"
    fi
    
    print_info "Building Docker image: $full_image_name"
    print_info "Platform: $PLATFORM"
    print_info "Dockerfile: $DOCKERFILE"
    
    # Prepare build command
    local build_cmd="docker build"
    
    # Add platform if specified
    if [[ "$PLATFORM" == *","* ]]; then
        print_info "Multi-platform build detected, using buildx"
        # Enable buildx for multi-platform builds
        docker buildx create --use --name expense-tracker-builder 2>/dev/null || true
        build_cmd="docker buildx build --platform $PLATFORM"
        
        if [[ "$PUSH" == true ]]; then
            build_cmd="$build_cmd --push"
        else
            build_cmd="$build_cmd --load"
        fi
    else
        build_cmd="$build_cmd --platform $PLATFORM"
    fi
    
    # Add build arguments
    if [[ -n "$BUILD_ARGS" ]]; then
        build_cmd="$build_cmd $BUILD_ARGS"
    fi
    
    # Add common build arguments
    build_cmd="$build_cmd --build-arg BUILDTIME=$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
    build_cmd="$build_cmd --build-arg VERSION=$TAG"
    build_cmd="$build_cmd --build-arg REVISION=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
    
    # Add tag and dockerfile
    build_cmd="$build_cmd -t $full_image_name -f $DOCKERFILE ."
    
    print_info "Executing: $build_cmd"
    
    # Execute build
    if eval "$build_cmd"; then
        print_success "Image built successfully: $full_image_name"
        
        # Show image size
        if [[ "$PLATFORM" != *","* ]]; then
            local image_size=$(docker images --format "table {{.Size}}" "$full_image_name" | tail -n 1)
            print_info "Image size: $image_size"
        fi
    else
        print_error "Failed to build image"
        exit 1
    fi
}

# Function to push Docker image
push_image() {
    if [[ -z "$REGISTRY" ]]; then
        print_warning "No registry specified, skipping push"
        return
    fi
    
    # Skip push if already pushed via buildx
    if [[ "$PLATFORM" == *","* ]]; then
        print_success "Image already pushed via buildx"
        return
    fi
    
    local full_image_name="$REGISTRY/$IMAGE_NAME:$TAG"
    
    print_info "Pushing image to registry: $full_image_name"
    
    if docker push "$full_image_name"; then
        print_success "Image pushed successfully: $full_image_name"
    else
        print_error "Failed to push image"
        exit 1
    fi
}

# Function to generate image metadata
generate_metadata() {
    local full_image_name
    
    if [[ -n "$REGISTRY" ]]; then
        full_image_name="$REGISTRY/$IMAGE_NAME:$TAG"
    else
        full_image_name="$IMAGE_NAME:$TAG"
    fi
    
    print_info "Generating image metadata..."
    
    cat > image-metadata.json << EOF
{
  "image": "$full_image_name",
  "tag": "$TAG",
  "registry": "$REGISTRY",
  "platform": "$PLATFORM",
  "buildTime": "$(date -u +'%Y-%m-%dT%H:%M:%SZ')",
  "revision": "$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')",
  "branch": "$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo 'unknown')",
  "dockerfile": "$DOCKERFILE"
}
EOF
    
    print_success "Metadata saved to image-metadata.json"
}

# Function to clean up
cleanup() {
    if [[ "$PLATFORM" == *","* ]]; then
        print_info "Cleaning up buildx builder..."
        docker buildx rm expense-tracker-builder 2>/dev/null || true
    fi
}

# Trap cleanup on exit
trap cleanup EXIT

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -r|--registry)
            REGISTRY="$2"
            shift 2
            ;;
        -i|--image)
            IMAGE_NAME="$2"
            shift 2
            ;;
        -t|--tag)
            TAG="$2"
            shift 2
            ;;
        -p|--platform)
            PLATFORM="$2"
            shift 2
            ;;
        -f|--dockerfile)
            DOCKERFILE="$2"
            shift 2
            ;;
        -b|--build-arg)
            BUILD_ARGS="$BUILD_ARGS --build-arg $2"
            shift 2
            ;;
        --push)
            PUSH=true
            shift
            ;;
        --multi-platform)
            PLATFORM="linux/amd64,linux/arm64"
            shift
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Main execution
main() {
    print_info "Starting ExpenseTracker build process..."
    
    # Check prerequisites
    check_prerequisites
    
    # Detect and configure registry
    detect_registry
    
    # Build image
    build_image
    
    # Push image if requested
    if [[ "$PUSH" == true ]]; then
        push_image
    fi
    
    # Generate metadata
    generate_metadata
    
    print_success "Build process completed successfully!"
    
    # Show final information
    local full_image_name
    if [[ -n "$REGISTRY" ]]; then
        full_image_name="$REGISTRY/$IMAGE_NAME:$TAG"
    else
        full_image_name="$IMAGE_NAME:$TAG"
    fi
    
    echo ""
    print_info "Final Image: $full_image_name"
    print_info "Platform: $PLATFORM"
    
    if [[ "$PUSH" == true && -n "$REGISTRY" ]]; then
        print_info "Image available at: $full_image_name"
        echo ""
        print_info "To deploy this image:"
        echo "docker run -d -p 8080:8080 $full_image_name"
    else
        echo ""
        print_info "To run this image locally:"
        echo "docker run -d -p 8080:8080 $full_image_name"
    fi
}

# Run main function
main
