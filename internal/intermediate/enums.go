package intermediate

type AccessType int

const (
	AccesSTypeInvalid AccessType = iota
	AccessTypePublic
	AccessTypePrivate
)

var accessTypeStrings = []string{
	"invalid",
	"public",
	"private",
}

func GetAccessTypeString(a AccessType) string {
	return accessTypeStrings[a]
}

func GetAccessType(s string) AccessType {
	switch s {
	case "public":
		return AccessTypePublic
	case "private":
		return AccessTypePrivate
	case "":
		return AccessTypePublic
	default:
		return AccesSTypeInvalid
	}
}

type ProseType int

const (
	ProseTypeInvalid ProseType = iota
	ProseTypeInt
	ProseTypeBool
	ProseTypeString
)

var proseTypeStrings = []string{
	"invalid",
	"int64",
	"bool",
	"string",
}

func GetProseTypeString(p ProseType) string {
	return proseTypeStrings[p]
}

func GetProseType(s string) ProseType {
	switch s {
	case "int":
		return ProseTypeInt
	case "in64":
		return ProseTypeInt
	case "bool":
		return ProseTypeBool
	case "string":
		return ProseTypeString
	default:
		return ProseTypeInvalid
	}
}
