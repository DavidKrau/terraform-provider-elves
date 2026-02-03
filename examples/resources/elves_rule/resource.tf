resource "elves_rule" "TestCEL" {
  ruletype      = "BINARY"
  policy        = "CEL"
  identifier    = "this is test identifier"
  custommessage = "thsi is test message"
  isdefault     = true
  celexpression = "aaabbbddd"
}