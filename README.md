# Terralink

Tired of manually changing your Terraform module sources to local paths during development? Terralink is a command-line tool that simplifies this process, allowing you to seamlessly switch between remote and local module dependencies.

By adding a simple comment directive to your module blocks, you can instruct Terralink to "load" your local modules for development or "unload" them to revert to the original remote sources. This is especially useful for developers who need to frequently test changes locally without altering their main Terraform configuration.

## Installation

You can install Terralink using go install:
```bash
go install github.com/segator/terralink@latest
```

Alternatively, you can download a specific version from the GitHub releases page.

## Usage

To use Terralink, first add a special comment directive to your Terraform module blocks. This directive tells Terralink where to find the local version of the module.
The format for the directive is: 

```hcl
# terralink: path=/path/to/your/local/module
```

Example:
```hcl
module "aws_managed" {
    # terralink: path=../local/aws/managed
    source  = "my-registry/managed/aws"
    version = "1.2.3"
    
    # ... other module configurations
}
```
Once your directives are in place, you can use the following commands to manage your module dependencies.

### Load Local Modules

This command scans a directory for Terralink directives and modifies the source attribute of the corresponding modules to point to the local path.
```bash
terralink load --dir=/path/to/your/terraform/project
```

### Unload Local Modules

This command reverts the changes made by load, restoring the original remote source for the modules.
```bash
terralink unload --dir=/path/to/your/terraform/project
```

### Check Module Status

This command checks the status of your modules and exits with a non-zero status code if any local modules are currently loaded. This is perfect for integrating into your CI/CD pipeline or Git hooks to prevent committing local development configurations.
```bash
terralink check --dir=/path/to/your/terraform/project
```

Pro-Tip: Add the check command to a pre-commit Git hook or your CI pipeline to ensure you don't accidentally commit code with local module paths.
