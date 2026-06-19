# Configure the RacterMX provider with an API key.
# The API key can also be set via the RACTERMX_API_KEY environment variable.
provider "ractermx" {
  api_key  = var.ractermx_api_key
  base_url = "https://ractermx.com" # Optional, this is the default
}

variable "ractermx_api_key" {
  type      = string
  sensitive = true
}
