package genv

type tEnv int

const (
	envUndefined tEnv = iota
	envDev
	envProd
)

func (e tEnv) String() string {
	return [...]string{"dev", "prod", "undefined"}[e]
}

func (e *tEnv) Scan(value string) error {
	switch value {
	case "dev":
		*e = envDev
	case "prod":
		*e = envProd
	default:
		*e = envUndefined
	}
	return nil
}
