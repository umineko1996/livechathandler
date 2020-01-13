package livechathandler

import "time"

const (
	// 2020/01/13
	// $1 = ¥100

	// $1.00 - $1.99
	Tier1 SuperChatTier = iota + 1
	// $2.00 - $4.99
	Tier2
	// $5.00 - $9.99
	Tier3
	// $10.00 - $19.99
	Tier4
	// $20.00 - $49.99
	Tier5
	// $50.00 - $99.99
	Tier6
	// $100.00 - $199.99
	Tier7
	// $200.00 - $299.99
	Tier8
	// $300.00 - $399.99
	Tier9
	// $400.00 - $499.99
	Tier10
	// $500.00
	Tier11
)

func (t SuperChatTier) Color() string {
	switch t {
	case Tier1:
		return "Blue"
	case Tier2:
		return "Light blue"
	case Tier3:
		return "Green"
	case Tier4:
		return "Yellow"
	case Tier5:
		return "Orange"
	case Tier6:
		return "Magenta"
	case Tier7:
		fallthrough
	case Tier8:
		fallthrough
	case Tier9:
		fallthrough
	case Tier10:
		fallthrough
	case Tier11:
		return "Red"
	default:
		return ""
	}
}

func (t SuperChatTier) Ticker() time.Duration {
	switch t {
	case Tier1:
		return 0 * time.Second
	case Tier2:
		return 0 * time.Second
	case Tier3:
		return 2 * time.Minute
	case Tier4:
		return 5 * time.Minute
	case Tier5:
		return 10 * time.Minute
	case Tier6:
		return 30 * time.Minute
	case Tier7:
		return 1 * time.Hour
	case Tier8:
		return 2 * time.Hour
	case Tier9:
		return 3 * time.Hour
	case Tier10:
		return 4 * time.Hour
	case Tier11:
		return 5 * time.Hour
	default:
		return 0 * time.Second
	}
}
