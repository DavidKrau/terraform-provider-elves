resource "elves_rule" "TestCEL" {
  ruletype      = "BINARY"
  policy        = "CEL"
  identifier    = "this is test identifier"
  custommessage = "thsi is test message"
  isdefault     = true
  celexpression = "aaabbbddd"
}

resource "elves_rule" "BlockedApp" {
  ruletype   = "TEAMID"
  policy     = "BLOCKLIST"
  identifier = "EQHXZ8M8AV"
  isdefault  = true

  # Shown by Santa in the block dialog on the Mac, keep it short
  custommessage = "This app is not allowed"

  # Shown on the Elves rules page the dialog's button opens. Leave description
  # out when it would only repeat custommessage: the page then shows that.
  description              = "This app is not approved for company Macs because it uploads files to an unapproved cloud service."
  remediation_instructions = "Use the approved alternative from Self Service."
}
