#!/bin/bash

# ExpenseTracker Deployment Script
# Usage: ./deploy.sh [environment] [options]

set -e

# Default values
ENVIRONMENT=${1:-production}
BUILD_FRESH=${2:-false}
DOMAIN=${DOMAIN:-expenses.yourdomain.com}
EMAIL=${EMAIL:-your-email@domain.com}
JWT_SECRET=${JWT_SECRET:-}

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

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to generate JWT secret
generate_jwt_secret() {
    if [ -z "$JWT_SECRET" ]; then
        JWT_SECRET=$(openssl rand -base64 32 2>/dev/null || head -c 32 /dev/urandom | base64)
        print_info "Generated JWT secret: $JWT_SECRET"
    fi
}

# Function to check prerequisites
check_prerequisites() {
    print_info "Checking prerequisites..."
    
    if ! command_exists docker; then
        print_error "Docker is not installed. Please install Docker first."
        exit 1
    fi
    
    if ! command_exists docker-compose; then
        print_error "Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi
    
    if ! docker info >/dev/null 2>&1; then
        print_error "Docker daemon is not running. Please start Docker first."
        exit 1
    fi
    
    print_success "Prerequisites check passed"
}

# Function to setup environment
setup_environment() {
    print_info "Setting up environment: $ENVIRONMENT"
    
    # Create environment file
    cat > .env << EOF
# ExpenseTracker Environment Configuration
ENVIRONMENT=$ENVIRONMENT
DOMAIN=$DOMAIN
EMAIL=$EMAIL
JWT_SECRET=$JWT_SECRET

# Database Configuration
DB_PATH=/app/data/expense_tracker.db
BACKUP_ENABLED=true
BACKUP_SCHEDULE=0 2 * * *
BACKUP_RETENTION=30

# Application Configuration
GIN_MODE=release
PORT=8080

# Security Configuration
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_DURATION=1m

# Logging Configuration
LOG_LEVEL=info
LOG_FORMAT=json
EOF
    
    print_success "Environment configuration created"
}

# Function to build application
build_application() {
    if [ "$BUILD_FRESH" = "true" ]; then
        print_info "Building application with fresh containers..."
        docker-compose -f deployment/docker/docker-compose.yml build --no-cache
    else
        print_info "Building application..."
        docker-compose -f deployment/docker/docker-compose.yml build
    fi
    
    print_success "Application built successfully"
}

# Function to deploy application
deploy_application() {
    print_info "Deploying application..."
    
    # Stop existing containers
    docker-compose -f deployment/docker/docker-compose.yml down
    
    # Start new containers
    docker-compose -f deployment/docker/docker-compose.yml up -d
    
    # Wait for health check
    print_info "Waiting for application to start..."
    sleep 10
    
    # Check if application is running
    if docker-compose -f deployment/docker/docker-compose.yml ps | grep -q "Up"; then
        print_success "Application deployed successfully"
    else
        print_error "Application failed to start"
        docker-compose -f deployment/docker/docker-compose.yml logs
        exit 1
    fi
}

# Function to setup SSL certificate
setup_ssl() {
    if [ "$ENVIRONMENT" = "production" ]; then
        print_info "Setting up SSL certificate for $DOMAIN"
        
        # Update docker-compose with domain
        sed -i "s/expenses.yourdomain.com/$DOMAIN/g" deployment/docker/docker-compose.yml
        sed -i "s/your-email@domain.com/$EMAIL/g" deployment/docker/docker-compose.yml
        
        print_success "SSL configuration updated"
    fi
}

# Function to setup monitoring
setup_monitoring() {
    print_info "Setting up monitoring..."
    
    # Create monitoring configuration
    mkdir -p monitoring
    
    cat > monitoring/prometheus.yml << EOF
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'expense-tracker'
    static_configs:
      - targets: ['expense-tracker:8080']
    metrics_path: '/metrics'
EOF

    cat > monitoring/docker-compose.monitoring.yml << EOF
version: '3.8'

services:
  prometheus:
    image: prom/prometheus:latest
    container_name: expense-tracker-prometheus
    restart: unless-stopped
    ports:
      - "9090:9090"
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'

  grafana:
    image: grafana/grafana:latest
    container_name: expense-tracker-grafana
    restart: unless-stopped
    ports:
      - "3000:3000"
    volumes:
      - grafana_data:/var/lib/grafana
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin123
      - GF_USERS_ALLOW_SIGN_UP=false

volumes:
  prometheus_data:
  grafana_data:

networks:
  default:
    external:
      name: expense-tracker-network
EOF
    
    print_success "Monitoring setup complete"
}

# Function to create backup script
create_backup_script() {
    print_info "Creating backup script..."
    
    mkdir -p scripts
    
    cat > scripts/backup.sh << 'EOF'
#!/bin/bash

# ExpenseTracker Backup Script

BACKUP_DIR="/app/backups"
DB_PATH="/app/data/expense_tracker.db"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/expense_tracker_$DATE.db"

# Create backup directory if it doesn't exist
mkdir -p "$BACKUP_DIR"

# Create backup
if sqlite3 "$DB_PATH" ".backup $BACKUP_FILE"; then
    # Compress backup
    gzip "$BACKUP_FILE"
    echo "Backup created: $BACKUP_FILE.gz"
    
    # Clean old backups (keep last 30 days)
    find "$BACKUP_DIR" -name "*.gz" -mtime +30 -delete
    echo "Old backups cleaned"
else
    echo "Backup failed"
    exit 1
fi
EOF
    
    chmod +x scripts/backup.sh
    print_success "Backup script created"
}

# Function to show deployment status
show_status() {
    print_info "Deployment Status:"
    echo "===================="
    docker-compose -f deployment/docker/docker-compose.yml ps
    echo ""
    
    print_info "Application URLs:"
    echo "Main Application: http://localhost:8080"
    if [ "$ENVIRONMENT" = "production" ]; then
        echo "Public URL: https://$DOMAIN"
        echo "Traefik Dashboard: https://traefik.$DOMAIN"
    fi
    echo ""
    
    print_info "Useful Commands:"
    echo "View logs: docker-compose -f deployment/docker/docker-compose.yml logs -f"
    echo "Stop app: docker-compose -f deployment/docker/docker-compose.yml down"
    echo "Restart app: docker-compose -f deployment/docker/docker-compose.yml restart"
    echo "Update app: ./deploy.sh $ENVIRONMENT true"
}

# Function to setup nginx reverse proxy (alternative to Traefik)
setup_nginx() {
    print_info "Setting up Nginx reverse proxy..."
    
    mkdir -p nginx
    
    cat > nginx/nginx.conf << EOF
events {
    worker_connections 1024;
}

http {
    upstream expense_tracker {
        server expense-tracker:8080;
    }

    server {
        listen 80;
        server_name $DOMAIN;
        
        location /.well-known/acme-challenge/ {
            root /var/www/certbot;
        }
        
        location / {
            return 301 https://\$server_name\$request_uri;
        }
    }

    server {
        listen 443 ssl http2;
        server_name $DOMAIN;

        ssl_certificate /etc/nginx/ssl/live/$DOMAIN/fullchain.pem;
        ssl_certificate_key /etc/nginx/ssl/live/$DOMAIN/privkey.pem;
        
        ssl_protocols TLSv1.2 TLSv1.3;
        ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512:ECDHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES256-GCM-SHA384;
        ssl_prefer_server_ciphers off;
        
        location / {
            proxy_pass http://expense_tracker;
            proxy_set_header Host \$host;
            proxy_set_header X-Real-IP \$remote_addr;
            proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto \$scheme;
            
            # WebSocket support
            proxy_http_version 1.1;
            proxy_set_header Upgrade \$http_upgrade;
            proxy_set_header Connection "upgrade";
        }
    }
}
EOF
    
    print_success "Nginx configuration created"
}

# Main deployment function
main() {
    print_info "Starting ExpenseTracker deployment..."
    print_info "Environment: $ENVIRONMENT"
    print_info "Domain: $DOMAIN"
    print_info "Build fresh: $BUILD_FRESH"
    
    # Generate JWT secret if not provided
    generate_jwt_secret
    
    # Run deployment steps
    check_prerequisites
    setup_environment
    setup_ssl
    build_application
    deploy_application
    create_backup_script
    
    if [ "$ENVIRONMENT" = "production" ]; then
        setup_monitoring
    fi
    
    show_status
    
    print_success "Deployment completed successfully!"
    print_info "Your ExpenseTracker application is now running."
    
    if [ "$ENVIRONMENT" = "production" ]; then
        print_warning "Don't forget to:"
        print_warning "1. Point your domain DNS to this server"
        print_warning "2. Update the admin password for Traefik dashboard"
        print_warning "3. Configure firewall rules"
        print_warning "4. Set up regular backups"
    fi
}

# Script entry point
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    echo "ExpenseTracker Deployment Script"
    echo ""
    echo "Usage: $0 [environment] [build_fresh]"
    echo ""
    echo "Arguments:"
    echo "  environment    Deployment environment (development|production) [default: production]"
    echo "  build_fresh    Build with fresh containers (true|false) [default: false]"
    echo ""
    echo "Environment Variables:"
    echo "  DOMAIN         Your domain name [default: expenses.yourdomain.com]"
    echo "  EMAIL          Your email for SSL certificates [default: your-email@domain.com]"
    echo "  JWT_SECRET     JWT secret key [auto-generated if not provided]"
    echo ""
    echo "Examples:"
    echo "  $0 production"
    echo "  $0 development true"
    echo "  DOMAIN=myexpenses.com EMAIL=me@myexpenses.com $0 production"
    exit 0
fi

# Run main function
main
