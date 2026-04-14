package commands

type Globals struct {
	Config    string `short:"c" help:"Path to conduit config file" type:"path" required:""`
	LogLevel  string `help:"Log level" enum:"debug,info,warn,error,fatal" default:"info"`
	LogFormat string `help:"Log format" enum:"json,text" default:"text"`
}
