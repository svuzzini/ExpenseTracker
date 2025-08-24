# 🚀 ExpenseTracker Deployment Guide

This comprehensive guide covers multiple deployment options for the ExpenseTracker application, from local development to enterprise-grade cloud deployments.

## 📋 Table of Contents

1. [Quick Start](#quick-start)
2. [Local Development](#local-development)
3. [Docker Deployment](#docker-deployment)
4. [Cloud Deployment](#cloud-deployment)
5. [Production Considerations](#production-considerations)
6. [Monitoring & Maintenance](#monitoring--maintenance)
7. [Troubleshooting](#troubleshooting)

## 🚀 Quick Start

### Prerequisites
- **Go 1.21+** for local development
- **Docker & Docker Compose** for containerized deployment
- **Git** for source code management

### 1-Minute Local Setup
```bash
# Clone the repository
git clone <your-repo-url>
cd ExpenseTracker

# Install dependencies
go mod download

# Run the application
go run main.go

# Access at http://localhost:8080
```

### 1-Minute Docker Setup
```bash
# Clone and navigate to project
git clone <your-repo-url>
cd ExpenseTracker

# Deploy with Docker Compose
docker-compose -f deployment/docker/docker-compose.yml up -d

# Access at http://localhost:8080
```

## 💻 Local Development

### Environment Setup

1. **Install Go 1.21+**
   ```bash
   # On macOS with Homebrew
   brew install go
   
   # On Ubuntu/Debian
   sudo apt-get install golang-go
   
   # On Windows
   # Download from https://golang.org/dl/
   ```

2. **Clone and Setup**
   ```bash
   git clone <your-repo-url>
   cd ExpenseTracker
   go mod download
   ```

3. **Environment Configuration**
   ```bash
   # Create .env file (optional)
   cat > .env << EOF
   JWT_SECRET=your-development-secret-key
   GIN_MODE=debug
   PORT=8080
   DB_PATH=expense_tracker_dev.db
   EOF
   ```

4. **Run Development Server**
   ```bash
   # Standard run
   go run main.go
   
   # With hot reload (install air first: go install github.com/cosmtrek/air@latest)
   air
   
   # With custom port
   PORT=3000 go run main.go
   ```

### Development Tools

**Recommended IDE Extensions:**
- **VS Code**: Go extension, REST Client
- **GoLand**: Built-in Go support
- **Vim/Neovim**: vim-go plugin

**Useful Commands:**
```bash
# Format code
go fmt ./...

# Run tests
go test ./...

# Run with race detection
go run -race main.go

# Build for production
go build -o expense-tracker main.go
```

## 🐳 Docker Deployment

### Single Container Deployment

1. **Build and Run**
   ```bash
   # Build the image
   docker build -f deployment/docker/Dockerfile -t expense-tracker .
   
   # Run the container
   docker run -d \
     --name expense-tracker \
     -p 8080:8080 \
     -v expense_data:/app/data \
     -e JWT_SECRET=your-secret-key \
     expense-tracker
   ```

### Docker Compose Deployment (Recommended)

1. **Basic Deployment**
   ```bash
   # Navigate to project root
   cd ExpenseTracker
   
   # Deploy with compose
   docker-compose -f deployment/docker/docker-compose.yml up -d
   ```

2. **Production Deployment with SSL**
   ```bash
   # Set environment variables
   export DOMAIN=expenses.yourdomain.com
   export EMAIL=your-email@domain.com
   export JWT_SECRET=$(openssl rand -base64 32)
   
   # Deploy with automatic SSL
   docker-compose -f deployment/docker/docker-compose.yml up -d
   ```

3. **Using the Deployment Script**
   ```bash
   # Make script executable
   chmod +x deployment/scripts/deploy.sh
   
   # Deploy to production
   DOMAIN=expenses.yourdomain.com EMAIL=admin@yourdomain.com ./deployment/scripts/deploy.sh production
   
   # Deploy for development
   ./deployment/scripts/deploy.sh development
   ```

### Docker Compose Services

The compose file includes:
- **ExpenseTracker App**: Main application container
- **Traefik**: Reverse proxy with automatic SSL
- **Backup Service**: Automated SQLite backups
- **Monitoring** (optional): Prometheus & Grafana

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `JWT_SECRET` | Secret key for JWT tokens | Auto-generated | Yes |
| `DOMAIN` | Your domain name | expenses.yourdomain.com | Yes (production) |
| `EMAIL` | Email for SSL certificates | your-email@domain.com | Yes (production) |
| `GIN_MODE` | Gin framework mode | release | No |
| `PORT` | Application port | 8080 | No |
| `DB_PATH` | Database file path | /app/data/expense_tracker.db | No |

## ☁️ Cloud Deployment

### AWS Deployment (Terraform)

1. **Prerequisites**
   ```bash
   # Install Terraform
   curl -fsSL https://apt.releases.hashicorp.com/gpg | sudo apt-key add -
   sudo apt-add-repository "deb [arch=amd64] https://apt.releases.hashicorp.com $(lsb_release -cs) main"
   sudo apt-get update && sudo apt-get install terraform
   
   # Install AWS CLI
   curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
   unzip awscliv2.zip
   sudo ./aws/install
   
   # Configure AWS credentials
   aws configure
   ```

2. **Deploy Infrastructure**
   ```bash
   cd deployment/cloud
   
   # Initialize Terraform
   terraform init
   
   # Plan deployment
   terraform plan -var="domain_name=expenses.yourdomain.com"
   
   # Apply infrastructure
   terraform apply -var="domain_name=expenses.yourdomain.com"
   ```

3. **Build and Push Docker Image**
   ```bash
   # Get ECR login token
   aws ecr get-login-password --region us-west-2 | docker login --username AWS --password-stdin <account-id>.dkr.ecr.us-west-2.amazonaws.com
   
   # Build and tag image
   docker build -f deployment/docker/Dockerfile -t expense-tracker .
   docker tag expense-tracker:latest <account-id>.dkr.ecr.us-west-2.amazonaws.com/expense-tracker:latest
   
   # Push to ECR
   docker push <account-id>.dkr.ecr.us-west-2.amazonaws.com/expense-tracker:latest
   ```

4. **Deploy to ECS**
   ```bash
   # Update ECS service to use new image
   aws ecs update-service --cluster expense-tracker --service expense-tracker --force-new-deployment
   ```

### AWS Services Used

- **ECS Fargate**: Serverless container hosting
- **Application Load Balancer**: Traffic distribution and SSL termination
- **EFS**: Persistent file storage for SQLite database
- **ECR**: Container image registry
- **Secrets Manager**: Secure storage for JWT secrets
- **CloudWatch**: Logging and monitoring
- **Route 53**: DNS management (manual setup required)

### Other Cloud Providers

#### **Digital Ocean**
```bash
# Deploy to Digital Ocean App Platform
doctl apps create --spec deployment/cloud/digitalocean-app.yml
```

#### **Google Cloud Run**
```bash
# Build and deploy to Cloud Run
gcloud builds submit --tag gcr.io/PROJECT-ID/expense-tracker
gcloud run deploy --image gcr.io/PROJECT-ID/expense-tracker --platform managed
```

#### **Heroku**
```bash
# Deploy to Heroku
heroku create your-expense-tracker
heroku buildpacks:set heroku/go
git push heroku main
```

## 🏭 Production Considerations

### Security Checklist

- [ ] **SSL/TLS**: Enable HTTPS with valid certificates
- [ ] **JWT Secret**: Use strong, randomly generated secret
- [ ] **Rate Limiting**: Configure appropriate limits
- [ ] **Firewall**: Restrict access to necessary ports only
- [ ] **Database**: Encrypt SQLite database file
- [ ] **Secrets**: Use proper secret management (not environment variables)
- [ ] **Updates**: Keep base images and dependencies updated

### Performance Optimization

1. **Database Optimization**
   ```sql
   -- Enable WAL mode for better concurrency
   PRAGMA journal_mode=WAL;
   
   -- Optimize SQLite settings
   PRAGMA synchronous=NORMAL;
   PRAGMA cache_size=10000;
   PRAGMA temp_store=memory;
   ```

2. **Application Tuning**
   ```bash
   # Set optimal Go runtime
   export GOMAXPROCS=2
   export GOMEMLIMIT=512MB
   ```

3. **Container Resources**
   ```yaml
   # Docker Compose resource limits
   services:
     expense-tracker:
       deploy:
         resources:
           limits:
             cpus: '1'
             memory: 512M
           reservations:
             cpus: '0.5'
             memory: 256M
   ```

### Backup Strategy

1. **Automated Backups**
   ```bash
   # SQLite backup script (included in Docker compose)
   sqlite3 expense_tracker.db ".backup backup_$(date +%Y%m%d_%H%M%S).db"
   ```

2. **Cloud Storage Backup**
   ```bash
   # Upload to AWS S3
   aws s3 cp backup.db s3://your-backup-bucket/$(date +%Y/%m/%d)/
   
   # Upload to Google Cloud Storage
   gsutil cp backup.db gs://your-backup-bucket/$(date +%Y/%m/%d)/
   ```

### Scaling Considerations

1. **Horizontal Scaling**
   - Use load balancer for multiple instances
   - Implement session affinity for WebSocket connections
   - Consider Redis for session storage

2. **Database Scaling**
   - Monitor SQLite performance
   - Consider PostgreSQL for high concurrency
   - Implement read replicas if needed

## 📊 Monitoring & Maintenance

### Health Checks

The application provides health check endpoints:
- `GET /health` - Basic health check
- `GET /health/database` - Database connectivity check
- `GET /metrics` - Prometheus metrics (if enabled)

### Log Management

1. **Container Logs**
   ```bash
   # View application logs
   docker-compose -f deployment/docker/docker-compose.yml logs -f expense-tracker
   
   # View specific service logs
   docker logs expense-tracker-app
   ```

2. **Log Rotation**
   ```bash
   # Configure Docker log rotation
   docker run --log-driver json-file --log-opt max-size=10m --log-opt max-file=3
   ```

### Monitoring Stack

If using the optional monitoring services:

1. **Prometheus**: Metrics collection at `http://localhost:9090`
2. **Grafana**: Dashboards at `http://localhost:3000` (admin/admin123)

### Maintenance Tasks

1. **Regular Updates**
   ```bash
   # Update application
   docker-compose -f deployment/docker/docker-compose.yml pull
   docker-compose -f deployment/docker/docker-compose.yml up -d
   
   # Update system packages
   sudo apt update && sudo apt upgrade
   ```

2. **Database Maintenance**
   ```bash
   # Vacuum SQLite database
   sqlite3 expense_tracker.db "VACUUM;"
   
   # Analyze database statistics
   sqlite3 expense_tracker.db "ANALYZE;"
   ```

3. **Security Updates**
   ```bash
   # Update Docker base images
   docker pull alpine:3.18
   docker-compose -f deployment/docker/docker-compose.yml build --no-cache
   ```

## 🔧 Troubleshooting

### Common Issues

1. **Application Won't Start**
   ```bash
   # Check logs
   docker-compose -f deployment/docker/docker-compose.yml logs expense-tracker
   
   # Verify environment variables
   docker-compose -f deployment/docker/docker-compose.yml config
   
   # Check file permissions
   ls -la data/
   ```

2. **Database Issues**
   ```bash
   # Check database file
   sqlite3 expense_tracker.db ".schema"
   
   # Verify database integrity
   sqlite3 expense_tracker.db "PRAGMA integrity_check;"
   
   # Fix database corruption
   sqlite3 expense_tracker.db ".recover" | sqlite3 new_database.db
   ```

3. **SSL Certificate Issues**
   ```bash
   # Check certificate status
   docker-compose -f deployment/docker/docker-compose.yml logs traefik
   
   # Verify domain DNS
   dig expenses.yourdomain.com
   
   # Test certificate
   curl -I https://expenses.yourdomain.com
   ```

4. **Performance Issues**
   ```bash
   # Monitor resource usage
   docker stats
   
   # Check application metrics
   curl http://localhost:8080/metrics
   
   # Analyze database performance
   sqlite3 expense_tracker.db ".eqp full"
   ```

### Debug Mode

Enable debug mode for troubleshooting:
```bash
# Set debug environment
export GIN_MODE=debug
export LOG_LEVEL=debug

# Run with verbose logging
go run main.go
```

### Support Resources

- **Application Logs**: Check container logs for error messages
- **Health Endpoints**: Use `/health` for basic diagnostics
- **Database Tools**: Use SQLite CLI for database inspection
- **Network Tools**: Use `curl`, `wget`, `dig` for connectivity testing

### Getting Help

1. **Check Documentation**: Review this guide and README.md
2. **Search Issues**: Look through existing GitHub issues
3. **Create Issue**: Submit detailed bug report with logs
4. **Community**: Join discussions in project forums

---

## 📝 Quick Reference

### Essential Commands
```bash
# Local development
go run main.go

# Docker build
docker build -f deployment/docker/Dockerfile -t expense-tracker .

# Docker Compose deploy
docker-compose -f deployment/docker/docker-compose.yml up -d

# View logs
docker-compose -f deployment/docker/docker-compose.yml logs -f

# Stop services
docker-compose -f deployment/docker/docker-compose.yml down

# Database backup
sqlite3 expense_tracker.db ".backup backup.db"

# SSL certificate check
openssl s_client -connect expenses.yourdomain.com:443 -servername expenses.yourdomain.com
```

### Port Reference
- **8080**: Application (HTTP)
- **443**: HTTPS (production)
- **80**: HTTP redirect (production)
- **8081**: Traefik dashboard
- **9090**: Prometheus (optional)
- **3000**: Grafana (optional)

### File Locations
- **Database**: `/app/data/expense_tracker.db`
- **Backups**: `/app/backups/`
- **Logs**: Docker container logs
- **SSL Certs**: Managed by Traefik/Let's Encrypt
- **Static Files**: `/app/static/`
- **Templates**: `/app/templates/`

---

🎉 **Your ExpenseTracker application is now ready for deployment!** Choose the method that best fits your needs and infrastructure requirements.
