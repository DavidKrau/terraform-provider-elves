# There is one set of FAA settings, declare it at most once
resource "elves_faa_settings" "this" {
  event_detail_url  = "https://elves.example.com/faa/?machine=%machine_id%"
  event_detail_text = "More info"
}
