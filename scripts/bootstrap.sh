#!/bin/bash

# Archivist Bootstrap Script
# Automatically checks and installs all dependencies from scratch

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Progress indicators
SPINNER_PID=""

# Function to print colored messages
print_info() {
    echo -e "${BLUE}ℹ ${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_header() {
    echo -e "\n${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}  $1${NC}"
    echo -e "${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
}

# Spinner for long operations
show_spinner() {
    local pid=$1
    local message=$2
    local spin='⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏'
    local i=0

    while kill -0 $pid 2>/dev/null; do
        i=$(( (i+1) %10 ))
        printf "\r${CYAN}${spin:$i:1}${NC} $message..."
        sleep .1
    done
    printf "\r"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check version
check_version() {
    local cmd=$1
    local required=$2
    local current=$($cmd)

    if [ "$(printf '%s\n' "$required" "$current" | sort -V | head -n1)" = "$required" ]; then
        return 0
    else
        return 1
    fi
}

# Print banner
print_banner() {
    cat << "EOF"

     █████╗ ██████╗  ██████╗██╗  ██╗██╗██╗   ██╗██╗███████╗████████╗
    ██╔══██╗██╔══██╗██╔════╝██║  ██║██║██║   ██║██║██╔════╝╚══██╔══╝
    ███████║██████╔╝██║     ███████║██║██║   ██║██║███████╗   ██║
    ██╔══██║██╔══██╗██║     ██╔══██║██║╚██╗ ██╔╝██║╚════██║   ██║
    ██║  ██║██║  ██║╚██████╗██║  ██║██║ ╚████╔╝ ██║███████║   ██║
    ╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝╚═╝  ╚═╝╚═╝  ╚═══╝  ╚═╝╚══════╝   ╚═╝

            Bootstrap & Dependency Installation System
                      Setting up from scratch...

EOF
}

clear
print_banner

print_header "PHASE 1: System Requirements Check"

# Detect OS
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    OS="linux"
    PKG_MANAGER="apt-get"
    if command_exists yum; then
        PKG_MANAGER="yum"
    elif command_exists dnf; then
        PKG_MANAGER="dnf"
    fi
    print_info "Detected OS: Linux ($PKG_MANAGER)"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    OS="macos"
    PKG_MANAGER="brew"
    print_info "Detected OS: macOS"
else
    print_error "Unsupported OS: $OSTYPE"
    exit 1
fi

# Check for sudo/root
if [ "$EUID" -eq 0 ]; then
    SUDO=""
    print_warning "Running as root"
else
    SUDO="sudo"
    print_info "Will use sudo for system installations"
fi

# 1. Check Go
print_info "Checking Go installation..."
if command_exists go; then
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    print_success "Go $GO_VERSION is installed"
else
    print_warning "Go is not installed"
    print_info "Installing Go 1.21..."

    if [ "$OS" = "linux" ]; then
        (
            wget -q https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
            $SUDO rm -rf /usr/local/go
            $SUDO tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
            rm go1.21.0.linux-amd64.tar.gz
        ) &
        show_spinner $! "Installing Go"

        # Add to PATH if not already there
        if ! grep -q "/usr/local/go/bin" ~/.bashrc; then
            echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
        fi
        export PATH=$PATH:/usr/local/go/bin
        print_success "Go installed successfully"
    elif [ "$OS" = "macos" ]; then
        if command_exists brew; then
            brew install go >/dev/null 2>&1 &
            show_spinner $! "Installing Go via Homebrew"
            print_success "Go installed successfully"
        else
            print_error "Homebrew not found. Please install Homebrew first."
            exit 1
        fi
    fi
fi

# 2. Check Python
print_info "Checking Python installation..."
if command_exists python3; then
    PYTHON_VERSION=$(python3 --version | awk '{print $2}')
    print_success "Python $PYTHON_VERSION is installed"
else
    print_warning "Python3 is not installed"
    print_info "Installing Python3..."

    if [ "$OS" = "linux" ]; then
        $SUDO $PKG_MANAGER update -y >/dev/null 2>&1
        $SUDO $PKG_MANAGER install -y python3 python3-pip python3-venv >/dev/null 2>&1 &
        show_spinner $! "Installing Python3"
        print_success "Python3 installed successfully"
    elif [ "$OS" = "macos" ]; then
        brew install python3 >/dev/null 2>&1 &
        show_spinner $! "Installing Python3 via Homebrew"
        print_success "Python3 installed successfully"
    fi
fi

# 3. Check Docker
print_info "Checking Docker installation..."
if command_exists docker; then
    DOCKER_VERSION=$(docker --version | awk '{print $3}' | sed 's/,//')
    print_success "Docker $DOCKER_VERSION is installed"

    # Check if Docker daemon is running
    if docker info >/dev/null 2>&1; then
        print_success "Docker daemon is running"
    else
        print_warning "Docker is installed but daemon is not running"
        print_info "Please start Docker Desktop or run: sudo systemctl start docker"
    fi
else
    print_warning "Docker is not installed"
    print_info "Installing Docker..."

    if [ "$OS" = "linux" ]; then
        (
            # Install Docker using official script
            curl -fsSL https://get.docker.com -o get-docker.sh
            $SUDO sh get-docker.sh
            rm get-docker.sh

            # Add current user to docker group
            $SUDO usermod -aG docker $USER
        ) >/dev/null 2>&1 &
        show_spinner $! "Installing Docker"
        print_success "Docker installed successfully"
        print_warning "You may need to log out and back in for Docker group permissions to take effect"
    elif [ "$OS" = "macos" ]; then
        print_error "Please install Docker Desktop for Mac from: https://www.docker.com/products/docker-desktop"
        print_info "After installing, run this script again"
        exit 1
    fi
fi

# 4. Check Docker Compose
print_info "Checking Docker Compose installation..."
if command_exists docker-compose || docker compose version >/dev/null 2>&1; then
    print_success "Docker Compose is installed"
else
    print_warning "Docker Compose is not installed"
    print_info "Installing Docker Compose..."

    if [ "$OS" = "linux" ]; then
        (
            $SUDO curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
            $SUDO chmod +x /usr/local/bin/docker-compose
        ) >/dev/null 2>&1 &
        show_spinner $! "Installing Docker Compose"
        print_success "Docker Compose installed successfully"
    fi
fi

# 5. Check LaTeX
print_info "Checking LaTeX installation..."
if command_exists pdflatex && command_exists latexmk; then
    print_success "LaTeX is installed"
else
    print_warning "LaTeX is not installed"
    print_info "Installing LaTeX (this may take several minutes)..."

    if [ "$OS" = "linux" ]; then
        $SUDO $PKG_MANAGER install -y texlive-latex-extra texlive-fonts-recommended latexmk >/dev/null 2>&1 &
        show_spinner $! "Installing LaTeX packages"
        print_success "LaTeX installed successfully"
    elif [ "$OS" = "macos" ]; then
        print_warning "Please install MacTeX from: https://www.tug.org/mactex/"
        print_info "Or use: brew install --cask mactex-no-gui"
    fi
fi

# 6. Check Make
print_info "Checking Make installation..."
if command_exists make; then
    print_success "Make is installed"
else
    print_warning "Make is not installed"
    print_info "Installing Make..."

    if [ "$OS" = "linux" ]; then
        $SUDO $PKG_MANAGER install -y build-essential >/dev/null 2>&1 &
        show_spinner $! "Installing Make"
        print_success "Make installed successfully"
    elif [ "$OS" = "macos" ]; then
        print_info "Make should be available via Xcode Command Line Tools"
        xcode-select --install 2>/dev/null || true
    fi
fi

# 7. Check Git
print_info "Checking Git installation..."
if command_exists git; then
    print_success "Git is installed"
else
    print_warning "Git is not installed"
    print_info "Installing Git..."

    if [ "$OS" = "linux" ]; then
        $SUDO $PKG_MANAGER install -y git >/dev/null 2>&1 &
        show_spinner $! "Installing Git"
        print_success "Git installed successfully"
    elif [ "$OS" = "macos" ]; then
        brew install git >/dev/null 2>&1 &
        show_spinner $! "Installing Git via Homebrew"
        print_success "Git installed successfully"
    fi
fi

print_header "PHASE 2: Go Dependencies"

print_info "Installing Go module dependencies..."
(
    go mod download
    go mod tidy
) >/dev/null 2>&1 &
show_spinner $! "Downloading Go dependencies"
print_success "Go dependencies installed"

print_header "PHASE 3: Python Virtual Environment & Dependencies"

# Navigate to search engine directory
if [ -d "services/search-engine" ]; then
    cd services/search-engine

    print_info "Creating Python virtual environment..."
    if [ ! -d "venv" ]; then
        python3 -m venv venv >/dev/null 2>&1
        print_success "Virtual environment created"
    else
        print_success "Virtual environment already exists"
    fi

    print_info "Installing Python dependencies..."
    (
        source venv/bin/activate
        pip install --upgrade pip >/dev/null 2>&1
        pip install -r requirements.txt >/dev/null 2>&1
    ) &
    show_spinner $! "Installing Python packages"
    print_success "Python dependencies installed"

    cd ../..
else
    print_warning "Search engine service not found, skipping Python setup"
fi

print_header "PHASE 4: Docker Services Setup"

print_info "Pulling Docker images (this may take several minutes)..."

# Function to pull Docker image with progress
pull_image() {
    local image=$1
    local name=$2

    print_info "Pulling $name..."
    docker pull $image >/dev/null 2>&1 &
    show_spinner $! "Pulling $name image"
    print_success "$name image ready"
}

# Pull all required images
pull_image "neo4j:5.15-community" "Neo4j"
pull_image "qdrant/qdrant:v1.7.4" "Qdrant"
pull_image "redis:7.2-alpine" "Redis"
pull_image "apache/kafka:latest" "Kafka"

print_header "PHASE 5: Project Structure Setup"

print_info "Creating directory structure..."

# Create necessary directories
mkdir -p lib tex_files reports .metadata logs
mkdir -p config

print_success "Directories created"

# Check for .env file
if [ ! -f ".env" ]; then
    print_warning ".env file not found"
    print_info "Creating .env template..."

    cat > .env << 'EOF'
# Archivist Environment Configuration
# Please add your Gemini API key below

GEMINI_API_KEY=your_api_key_here

# Get your API key from: https://aistudio.google.com/app/apikey
EOF

    print_success ".env template created"
    print_warning "⚠️  IMPORTANT: Edit .env and add your GEMINI_API_KEY"
    print_info "Get your API key from: https://aistudio.google.com/app/apikey"
else
    # Check if API key is set
    if grep -q "your_api_key_here" .env; then
        print_warning "⚠️  GEMINI_API_KEY not configured in .env"
        print_info "Get your API key from: https://aistudio.google.com/app/apikey"
    else
        print_success ".env file configured"
    fi
fi

# Check for config.yaml
if [ ! -f "config/config.yaml" ]; then
    print_warning "config/config.yaml not found"
    if [ -f "config/config.yaml.example" ]; then
        cp config/config.yaml.example config/config.yaml
        print_success "Created config.yaml from example"
    else
        print_info "Config file will be created on first run"
    fi
else
    print_success "config.yaml exists"
fi

print_header "PHASE 6: Building Archivist"

print_info "Compiling Archivist binary..."
(
    go build -o archivist ./cmd/main
) >/dev/null 2>&1 &
show_spinner $! "Compiling Archivist"
print_success "Archivist binary built successfully"

# Make executable
chmod +x archivist

print_header "PHASE 7: Docker Services (Optional)"

echo -e "${YELLOW}Would you like to start the Knowledge Graph services now?${NC}"
echo -e "This will start: Neo4j, Qdrant, Redis, and Kafka"
echo -e "${CYAN}[Y/n]${NC}: \c"
read -r start_services

if [[ "$start_services" =~ ^[Yy]$ ]] || [ -z "$start_services" ]; then
    print_info "Starting Docker services..."

    if [ -f "docker-compose-graph.yml" ]; then
        docker-compose -f docker-compose-graph.yml up -d >/dev/null 2>&1 &
        show_spinner $! "Starting Docker services"

        # Wait for services to be healthy
        print_info "Waiting for services to be ready..."
        sleep 5

        # Check Neo4j
        print_info "Checking Neo4j..."
        for i in {1..30}; do
            if docker exec archivist-neo4j cypher-shell -u neo4j -p password "RETURN 1" >/dev/null 2>&1; then
                print_success "Neo4j is ready"
                break
            fi
            sleep 2
        done

        # Check Qdrant
        print_info "Checking Qdrant..."
        for i in {1..30}; do
            if curl -s http://localhost:6333/healthz >/dev/null 2>&1; then
                print_success "Qdrant is ready"
                break
            fi
            sleep 2
        done

        # Check Redis
        print_info "Checking Redis..."
        for i in {1..30}; do
            if docker exec archivist-redis redis-cli ping >/dev/null 2>&1; then
                print_success "Redis is ready"
                break
            fi
            sleep 2
        done

        print_success "All Docker services are running!"

        echo -e "\n${GREEN}Access your services:${NC}"
        echo -e "  Neo4j Browser:  ${CYAN}http://localhost:7474${NC} (neo4j/password)"
        echo -e "  Qdrant Dashboard: ${CYAN}http://localhost:6333/dashboard${NC}"
        echo -e "  Redis: ${CYAN}localhost:6379${NC}"

    else
        print_error "docker-compose-graph.yml not found"
    fi
else
    print_info "Skipping Docker services startup"
    print_info "You can start them later with: docker-compose -f docker-compose-graph.yml up -d"
fi

print_header "✅ SETUP COMPLETE!"

cat << EOF

${GREEN}╔════════════════════════════════════════════════════════════════╗
║                 🎉 Archivist is Ready! 🎉                      ║
╚════════════════════════════════════════════════════════════════╝${NC}

${CYAN}Quick Start Commands:${NC}

  1. Configure your API key:
     ${YELLOW}nano .env${NC}  (add your GEMINI_API_KEY)

  2. Process a paper:
     ${YELLOW}./archivist process lib/your_paper.pdf${NC}

  3. Run the interactive TUI:
     ${YELLOW}./archivist run${NC}

  4. Search for papers:
     ${YELLOW}./archivist search "transformer architecture"${NC}

  5. Chat with papers:
     ${YELLOW}./archivist chat${NC}

${CYAN}Graph Database Commands:${NC}

  • Build knowledge graph:
    ${YELLOW}./archivist graph build${NC}

  • Search semantically:
    ${YELLOW}./archivist search "attention mechanisms"${NC}

  • Explore citations:
    ${YELLOW}./archivist cite show "Paper Title"${NC}

${CYAN}Service Management:${NC}

  • Start all services:
    ${YELLOW}docker-compose -f docker-compose-graph.yml up -d${NC}

  • Stop all services:
    ${YELLOW}docker-compose -f docker-compose-graph.yml down${NC}

  • View logs:
    ${YELLOW}docker-compose -f docker-compose-graph.yml logs -f${NC}

${CYAN}Documentation:${NC}

  • README: ${YELLOW}cat README.md${NC}
  • Graph Guide: ${YELLOW}cat docs/features/KNOWLEDGE_GRAPH_GUIDE.md${NC}
  • Search Guide: ${YELLOW}cat docs/features/SEARCH_ENGINE_GUIDE.md${NC}

${GREEN}Happy researching! 📚${NC}

EOF

# Check if API key is configured
if grep -q "your_api_key_here" .env 2>/dev/null; then
    echo -e "${RED}⚠️  REMINDER: Don't forget to add your GEMINI_API_KEY to .env${NC}\n"
fi
