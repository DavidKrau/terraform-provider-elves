resource "elves_configuration" "engineering" {
  name        = "Engineering"
  client_mode = "LOCKDOWN"
  sync_type   = "CLEAN_ALL"

  # USB storage read-only and not executable
  block_usb_mount  = true
  remount_usb_mode = "noexec,rdonly"
}

# Rules for that configuration
resource "elves_config_rule" "engineering_tools" {
  rule_id    = elves_rule.docker.id
  config_ids = [elves_configuration.engineering.id]
}
