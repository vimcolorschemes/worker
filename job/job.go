package job

type JobType string

const (
	Import JobType = "import"
	Update JobType = "update"
	Clean  JobType = "clean"
)

var Jobs = []JobType{Import, Update, Clean}

func GetJob(args []string) JobType {
	if len(args) < 2 {
		return Import
	}

	switch JobType(args[1]) {
	case Import:
		return Import
	case Update:
		return Update
	case Clean:
		return Clean
	default:
		return Import
	}
}
