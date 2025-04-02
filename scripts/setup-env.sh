#!/bin/bash

# Function to copy template if file doesn't exist
copy_template() {
    local template=$1
    local target=$2
    if [ ! -f "$target" ]; then
        cp "$template" "$target"
        echo "Created $target from template"
    else
        echo "$target already exists, skipping..."
    fi
}

# Create config files from templates
copy_template configs/api.yaml.template configs/api.yaml
copy_template configs/operator.yaml.template configs/operator.yaml
copy_template configs/cli.yaml.template configs/cli.yaml

# Make sure the files are readable only by the owner
chmod 600 configs/*.yaml 2>/dev/null

echo "Configuration files have been set up. Please edit them with your specific values."
echo "Remember to never commit the actual config files to version control!" 