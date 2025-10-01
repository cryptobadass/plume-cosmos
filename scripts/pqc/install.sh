#!/bin/bash

# Falcon PQC Universal Installation Script
# Auto-detect operating system and run corresponding installation script

echo "🚀 Falcon PQC Universal Installation Script"
echo "============================================"

# Detect operating system
if [[ "$OSTYPE" == "darwin"* ]]; then
    echo "🍎 Detected macOS system"
    SCRIPT_DIR="scripts/pqc/mac"
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    echo "🐧 Detected Linux system"
    SCRIPT_DIR="scripts/pqc/ubuntu"
else
    echo "❌ Error: Unsupported operating system: $OSTYPE"
    echo "Supported systems: macOS, Ubuntu/Debian"
    exit 1
fi

# Check if script exists
if [ ! -f "$SCRIPT_DIR/quick_setup.sh" ]; then
    echo "❌ Error: Cannot find $SCRIPT_DIR/quick_setup.sh"
    exit 1
fi

# Add execution permissions to scripts
chmod +x "$SCRIPT_DIR"/*.sh

# Run corresponding installation script
echo "📦 Running $SCRIPT_DIR/quick_setup.sh..."
"$SCRIPT_DIR/quick_setup.sh"

# Set environment variables
echo ""
echo "⚙️  Setting environment variables..."

# Get the project directory
PROJECT_DIR=$(pwd)

if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS environment variable setup
    if [[ -d "/opt/homebrew/lib" ]]; then
        # Apple Silicon Mac
        echo "export CGO_CFLAGS=\"-I/opt/homebrew/include\"" >> ~/.zshrc
        echo "export CGO_LDFLAGS=\"-L/opt/homebrew/lib -loqs -lssl -lcrypto\"" >> ~/.zshrc
        echo "export PKG_CONFIG_PATH=\"\$PKG_CONFIG_PATH:$HOME/liboqs-go/.config\"" >> ~/.zshrc
        echo "export DYLD_LIBRARY_PATH=\"\$DYLD_LIBRARY_PATH:/opt/homebrew/lib\"" >> ~/.zshrc
    else
        # Intel Mac
        echo "export CGO_CFLAGS=\"-I/usr/local/include\"" >> ~/.zshrc
        echo "export CGO_LDFLAGS=\"-L/usr/local/lib -loqs -lssl -lcrypto\"" >> ~/.zshrc
        echo "export PKG_CONFIG_PATH=\"\$PKG_CONFIG_PATH:$HOME/liboqs-go/.config\"" >> ~/.zshrc
        echo "export DYLD_LIBRARY_PATH=\"\$DYLD_LIBRARY_PATH:/usr/local/lib\"" >> ~/.zshrc
    fi
    echo "✅ macOS environment variables set to ~/.zshrc"
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    # Ubuntu environment variable setup
    echo "export CGO_CFLAGS=\"-I/usr/local/include\"" >> ~/.bashrc
    echo "export CGO_LDFLAGS=\"-L/usr/local/lib -loqs -lssl -lcrypto\"" >> ~/.bashrc
    echo "export PKG_CONFIG_PATH=\"\$PKG_CONFIG_PATH:$PROJECT_DIR/.config\"" >> ~/.bashrc
    echo "export LD_LIBRARY_PATH=\"\$LD_LIBRARY_PATH:/usr/local/lib\"" >> ~/.bashrc
    echo "✅ Ubuntu environment variables set to ~/.bashrc"
fi

# Reload environment variables
echo ""
echo "🔄 Reloading environment variables..."
if [[ "$OSTYPE" == "darwin"* ]]; then
    # Use zsh to properly load Oh My Zsh and environment variables
    if command -v zsh &> /dev/null; then
        zsh -c "source ~/.zshrc && echo '✅ macOS environment variables reloaded'"
    else
        echo "⚠️  zsh not found, environment variables will be available in new terminal sessions"
    fi
else
    source ~/.bashrc
    echo "✅ Ubuntu environment variables reloaded"
fi

echo ""
echo "✅ Installation completed!"
