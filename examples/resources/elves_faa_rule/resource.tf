# Only Slack (and the default processes) may read or write Slack's cookies
resource "elves_faa_rule" "slack_cookies" {
  name      = "SlackCookies"
  rule_type = "PathsWithAllowedProcesses"

  paths = [
    { path = "/Users/*/Library/Application Support/Slack/Cookies", is_prefix = true },
    { path = "/Users/*/Library/Containers/com.tinyspeck.slackmacgap/Data/Library/Application Support/Slack/Cookies", is_prefix = true },
  ]

  # The default processes of elves_faa_settings (EDR, Spotlight) are added too
  processes = [
    { team_id = "BQR82RBBHL", signing_id = "com.tinyspeck.slackmacgap*" },
  ]

  # Block, not only log, other processes. Defaults to true.
  audit_only = false

  block_message = "Only Slack may access Slack's cookies."
  description   = "Slack's cookies give access to your Slack account, so other apps may not read them."
}
