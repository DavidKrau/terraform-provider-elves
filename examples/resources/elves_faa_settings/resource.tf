# There is one set of FAA settings, declare it at most once
resource "elves_faa_settings" "this" {
  event_detail_url  = "https://elves.example.com/v1/www/rules/?machine=%machine_id%&rule=%rule_name%"
  event_detail_text = "Details"

  # Added to every PathsWithAllowedProcesses rule, so they aren't repeated in each
  default_processes = [
    # EDR agent
    { team_id = "4AYE5J54KN", signing_id = "com.sentinelone.sentineld-helper" },
    # Spotlight
    { signing_id = "com.apple.mds", platform_binary = true },
    { signing_id = "platform:com.apple.mds", platform_binary = true },
    { signing_id = "com.apple.mdworker_shared", platform_binary = true },
  ]
}
