#!/bin/bash
# Setup a single node chain based on chain.md documentation
# This script automates the process described in the documentation

set -e

# Configuration variables
MONIKER="node1"
CHAIN_ID="test-chain"
KEYNAME="admin"
AMOUNT="1000000000000000000000000000uplume"
GENTX_AMOUNT="7000000000000000uplume"
KEYRING_BACKEND="test"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m' 
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
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

# Function to validate inputs
validate_inputs() {
    if [ -z "$MONIKER" ]; then
        print_error "MONIKER cannot be empty. Please set a valid moniker."
        exit 1
    fi
    
    if [ -z "$CHAIN_ID" ]; then
        print_error "CHAIN_ID cannot be empty. Please set a valid chain ID."
        exit 1
    fi
}

# Function to configure keyring backend
configure_keyring_backend() {
    print_status "Configuring keyring backend: $KEYRING_BACKEND"
    ./build/simd config keyring-backend "$KEYRING_BACKEND" --home ~/.simapp
    print_status "Keyring backend configured successfully."
}

# Function to build from source
build_from_source() {
    print_status "Building from source..."
    
    if [ ! -d ".git" ]; then
        print_error "Not in a git repository. Please run this script from the plume-cosmos directory."
        exit 1
    fi
    
    if ! command_exists make; then
        print_error "make command not found. Please install build tools."
        exit 1
    fi
    
    make build
    print_status "Build completed successfully."
}

# Function to initialize node
initialize_node() {
    print_status "Initializing node with moniker: $MONIKER and chain-id: $CHAIN_ID"
    
    # Clean up old configuration if exists
    if [ -d "$HOME/.simapp" ]; then
        print_warning "Removing existing .simapp directory..."
        rm -rf "$HOME/.simapp"
    fi
    
    ./build/simd init "$MONIKER" --chain-id "$CHAIN_ID" --home ~/.simapp
    print_status "Node initialization completed."
}

# Function to create key
create_key() {
    print_status "Creating key: $KEYNAME with PQC algorithm"
    echo -e "\n\n" | ./build/simd keys add "$KEYNAME" --algo falcon-512 --keyring-backend "$KEYRING_BACKEND" --home ~/.simapp
    print_status "Key created successfully."
}

# Function to add genesis account
add_genesis_account() {
    print_status "Adding genesis account..."
    
    # Get the account address
    ACCOUNT_ADDR=$(./build/simd keys show "$KEYNAME" -a --keyring-backend "$KEYRING_BACKEND" --home ~/.simapp)
    print_status "Account address: $ACCOUNT_ADDR"
    
    # Add genesis account
    ./build/simd add-genesis-account "$ACCOUNT_ADDR" "$AMOUNT" --home ~/.simapp
    print_status "Genesis account added successfully."
}

# Function to create additional accounts
create_additional_accounts() {
    print_status "Creating additional accounts (ta0, ta1)..."
    
    # Create ta0 account
    print_status "Creating ta0 account with PQC algorithm..."
    echo -e "\n\n" | ./build/simd keys add ta0 --algo falcon-512 --keyring-backend "$KEYRING_BACKEND" --home ~/.simapp
    TA0_ADDR=$(./build/simd keys show ta0 -a --keyring-backend "$KEYRING_BACKEND" --home ~/.simapp)
    print_status "ta0 account address: $TA0_ADDR"
    
    # Create ta1 account
    print_status "Creating ta1 account with PQC algorithm..."
    echo -e "\n\n" | ./build/simd keys add ta1 --algo falcon-512 --keyring-backend "$KEYRING_BACKEND" --home ~/.simapp
    TA1_ADDR=$(./build/simd keys show ta1 -a --keyring-backend "$KEYRING_BACKEND" --home ~/.simapp)
    print_status "ta1 account address: $TA1_ADDR"
    
    # Add ta0 to genesis
    print_status "Adding ta0 to genesis..."
    ./build/simd add-genesis-account "$TA0_ADDR" 1000000000000000000000uplume --home ~/.simapp
    
    # Add ta1 to genesis
    print_status "Adding ta1 to genesis..."
    ./build/simd add-genesis-account "$TA1_ADDR" 1000000000000000000000uplume --home ~/.simapp
    
    print_status "Additional accounts created and added to genesis successfully."
}

# Function to generate genesis transaction
generate_gentx() {
    print_status "Generating genesis transaction..."
    
    NODE_ID=$(./build/simd tendermint show-node-id --home ~/.simapp)
    print_status "Node ID: $NODE_ID"
    
    # Get the validator's public key (ed25519 for Tendermint consensus)
    VALIDATOR_PUBKEY=$(./build/simd tendermint show-validator --home ~/.simapp)
    print_status "Validator pubkey: $VALIDATOR_PUBKEY"
    
    ./build/simd gentx "$KEYNAME" "$GENTX_AMOUNT" --node-id "$NODE_ID" --chain-id "$CHAIN_ID" --keyring-backend "$KEYRING_BACKEND" --home ~/.simapp
    print_status "Genesis transaction generated successfully."
}

# Function to add validator info to genesis
add_validator_info() {
    print_status "Adding validator info to genesis..."
    
    # Create add_validator.sh script
    cat > add_validator.sh << 'EOF'
#!/bin/bash
jq '.validators = []' ~/.simapp/config/genesis.json > ~/.simapp/config/tmp_genesis.json
cd ~/.simapp/config/gentx
IDX=0
for FILE in *
do
    jq '.validators['$IDX'] |= .+ {}' ~/.simapp/config/tmp_genesis.json > ~/.simapp/config/tmp_genesis_step_1.json && rm ~/.simapp/config/tmp_genesis.json
    KEY=$(jq '.body.messages[0].pubkey.key' $FILE -c)
    DELEGATION=$(jq -r '.body.messages[0].value.amount' $FILE)
    POWER=$(($DELEGATION / 1000000))
    jq '.validators['$IDX'] += {"power":"'$POWER'"}' ~/.simapp/config/tmp_genesis_step_1.json > ~/.simapp/config/tmp_genesis_step_2.json && rm ~/.simapp/config/tmp_genesis_step_1.json
    jq '.validators['$IDX'] += {"pub_key":{"type":"tendermint/PubKeyEd25519","value":'$KEY'}}' ~/.simapp/config/tmp_genesis_step_2.json > ~/.simapp/config/tmp_genesis_step_3.json && rm ~/.simapp/config/tmp_genesis_step_2.json
    mv ~/.simapp/config/tmp_genesis_step_3.json ~/.simapp/config/tmp_genesis.json
    IDX=$(($IDX+1))
done
mv ~/.simapp/config/tmp_genesis.json ~/.simapp/config/genesis.json
EOF
    
    chmod +x add_validator.sh
    ./add_validator.sh
    rm add_validator.sh
    print_status "Validator info added to genesis."
}

# Function to collect genesis transactions
collect_gentxs() {
    print_status "Collecting genesis transactions..."
    ./build/simd collect-gentxs --home ~/.simapp
    print_status "Genesis transactions collected successfully."
}

# Function to configure config.toml
configure_config_toml() {
    print_status "Configuring config.toml..."
    
    CONFIG_FILE="$HOME/.simapp/config/config.toml"
    CLIENT_FILE="$HOME/.simapp/config/client.toml"
    
    # Set chain-id in client.toml
    sed -i.bak 's/^chain-id = ""/chain-id = "test-chain"/' "$CLIENT_FILE"
    
    # Set mode to validator
    sed -i.bak 's/mode = "full"/mode = "validator"/' "$CONFIG_FILE"
    
    # Configure RPC
    sed -i.bak 's/laddr = "tcp:\/\/127.0.0.1:26657"/laddr = "tcp:\/\/0.0.0.0:26657"/' "$CONFIG_FILE"
    
    # Configure for single node validator
    sed -i.bak 's/create_empty_blocks = false/create_empty_blocks = true/' "$CONFIG_FILE"
    sed -i.bak 's/create_empty_blocks_interval = "0s"/create_empty_blocks_interval = "10s"/' "$CONFIG_FILE"
    sed -i.bak 's/timeout_commit = "1s"/timeout_commit = "2s"/' "$CONFIG_FILE"
    
    print_status "config.toml configured successfully."
}

# Function to configure app.toml
configure_app_toml() {
    print_status "Configuring app.toml..."
    
    APP_FILE="$HOME/.simapp/config/app.toml"
    
    # Set minimum gas prices
    sed -i.bak 's/minimum-gas-prices = ""/minimum-gas-prices = "0.025uplume"/' "$APP_FILE"
    
    print_status "app.toml configured successfully."
}

# Function to configure genesis.json
configure_genesis() {
    print_status "Configuring genesis.json..."
    
    GENESIS_FILE="$HOME/.simapp/config/genesis.json"
    
    # Set bond_denom and mint_denom to uplume
    jq '.app_state.staking.params.bond_denom = "uplume"' "$GENESIS_FILE" > "$GENESIS_FILE.tmp" && mv "$GENESIS_FILE.tmp" "$GENESIS_FILE"
    jq '.app_state.mint.params.mint_denom = "uplume"' "$GENESIS_FILE" > "$GENESIS_FILE.tmp" && mv "$GENESIS_FILE.tmp" "$GENESIS_FILE"
    jq '.app_state.crisis.constant_fee.denom = "uplume"' "$GENESIS_FILE" > "$GENESIS_FILE.tmp" && mv "$GENESIS_FILE.tmp" "$GENESIS_FILE"
    
    print_status "genesis.json configured successfully."
}

# Function to start node
start_node() {
    print_status "Starting node..."
    
    # Check if node is already running
    if pgrep -f "simd start" > /dev/null; then
        print_warning "Node is already running. Stopping existing node..."
        pkill -f "simd start" || true
        sleep 2
    fi
    
    # Start node in background
    nohup ./build/simd start --home ~/.simapp > simd.log 2>&1 &
    NODE_PID=$!
    
    print_status "Node started with PID: $NODE_PID"
    print_status "Logs are being written to: simd.log"
    
    # Wait a moment for node to start
    sleep 10
    
    # Check if node is running
    if pgrep -f "simd start" > /dev/null; then
        print_status "Node is running successfully!"
        print_status "You can check the status with: ./build/simd status --home ~/.simapp"
        print_status "You can view logs with: tail -f simd.log"
    else
        print_error "Failed to start node. Check simd.log for details."
        tail -20 simd.log
        exit 1
    fi
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -m, --moniker MONIKER     Set the moniker (node name, default: node1)"
    echo "  -c, --chain-id CHAIN_ID   Set the chain ID (default: test-chain)"
    echo "  -k, --keyname KEYNAME     Set the key name (default: admin)"
    echo "  -a, --amount AMOUNT       Set the genesis account amount (default: 1000000000000000000000000000uplume)"
    echo "  -g, --gentx-amount AMOUNT Set the gentx amount (default: 7000000000000000uplume)"
    echo "  -r, --keyring-backend     Set the keyring backend (test/file/os, default: test)"
    echo "  -h, --help               Show this help message"
    echo ""
    echo "Environment variables:"
    echo "  MONIKER                  Node moniker (default: node1)"
    echo "  CHAIN_ID                 Chain ID (default: test-chain)"
    echo "  KEYNAME                  Key name (default: admin)"
    echo "  AMOUNT                   Genesis account amount"
    echo "  GENTX_AMOUNT             Gentx amount"
    echo "  KEYRING_BACKEND          Keyring backend (default: test)"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Use all defaults"
    echo "  $0 -m \"my-node\" -c \"my-chain\"        # Override moniker and chain-id"
    echo "  $0 -r file                           # Use file keyring backend"
}

# Main function
main() {
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -m|--moniker)
                MONIKER="$2"
                shift 2
                ;;
            -c|--chain-id)
                CHAIN_ID="$2"
                shift 2
                ;;
            -k|--keyname)
                KEYNAME="$2"
                shift 2
                ;;
            -a|--amount)
                AMOUNT="$2"
                shift 2
                ;;
            -g|--gentx-amount)
                GENTX_AMOUNT="$2"
                shift 2
                ;;
            -r|--keyring-backend)
                KEYRING_BACKEND="$2"
                shift 2
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
    
    # Check if simd binary exists
    if [ ! -f "./build/simd" ]; then
        print_error "simd binary not found. Please run make build first."
        exit 1
    fi
    
    # Validate inputs
    validate_inputs
    
    print_status "Starting single node chain setup..."
    print_status "Moniker: $MONIKER"
    print_status "Chain ID: $CHAIN_ID"
    print_status "Key name: $KEYNAME"
    
    # Execute setup steps
    build_from_source
    
    initialize_node
    create_key
    add_genesis_account
    create_additional_accounts
    generate_gentx
    add_validator_info
    collect_gentxs
    configure_config_toml
    configure_app_toml
    configure_genesis
    configure_keyring_backend
    start_node
    
    print_status "Single node chain setup completed successfully!"
    print_status "Your node is now running. You can interact with it using ./build/simd commands."
}

# Run main function with all arguments
main "$@"

