# structarg

Argument parser for go that defines and parses command line arguements into a struct type variable. The attributes of arguments are defined in the comment tags of the struct. after parsing command-line argument or configuration files, the values are stored in this struct.

please refer to examples/example.go for example usages.

An example of a struct definition is shown as follows:

```go
type Options struct {
    Help bool       `help:"Show help messages" short-token:"h"`
    Debug bool      `help:"Show extra debug information"`
    Timeout int     `default:"600" help:"Number of seconds to wait for a response"`
    SUBCOMMAND string `help:"subcommand" subcommand:"true"`
}
```

## Argument name

Each member variable of the struct represents an argument. The variable name is the argument name. 

## Positional and optional arguments

If the variable name is all uppercased, the argument is a positional argument, otherwise, it is an optional argument. Additionally, boolean tag "optional" explicitly defines whether the argument is optional or positional.

## Tags

The attributes of an argument are defined in the comment tags of the member variable of the struct. The following tags are supported:

```go
	/*
	   help text of the argument
	   the argument is optional.
	*/
	TAG_HELP = "help"
	/*
	   command-line token for the optional argument, e.g. token:"url"
	   the command-line argument will be "--url http://127.0.0.1:3306"
	   the tag is optional.
	   if the tag is missing, the variable name will be used as token.
	   If the variable name is CamelCase, the token will be transformed
	   into kebab-case, e.g. if the variable is "AuthURL", the token will
	   be "--auth-url"
	*/
	TAG_TOKEN = "token"
	/*
	   short form of command-line token, e.g. short-token:"u"
	   the command-line argument will be "-u http://127.0.0.1:3306"
	   the tag is optional
	*/
	TAG_SHORT_TOKEN = "short-token"
	/*
	   Metavar of the argument
	   the tag is optional
	*/
	TAG_METAVAR = "metavar"
	/*
	   The default value of the argument.
	   the tag is optional
	*/
	TAG_DEFAULT = "default"
	/*
	   The possible values of an arguments. All choices are are concatenatd by "|".
	   e.g. `choices:"1|2|3"`
	   the tag is optional
	*/
	TAG_CHOICES = "choices"
	/*
	   A boolean value explicitly declare whether the argument is optional,
	   the tag is optional
	*/
	TAG_OPTIONAL = "optional"
	/*
	   A boolean value explicitly decalre whther the argument is an subcommand
	   A subcommand argument must be the last positional argument.
	   the tag is optional, the default value is false
	*/
	TAG_SUBCOMMAND = "subcommand"
	/*
	   The attribute defines the possible number of argument. Possible values
	   are:
	       * positive integers, e.g. "1", "2"
	       * "*" any number of arguments
	       * "+" at lease one argument
	       * "?" at most one argument
	   the tag is optional, the default value is "1"
	*/
	TAG_NARGS = "nargs"
	/*
		Alias name of argument
	*/
	TAG_ALIAS = "alias"
```

## Example usage

# use ParseArgs which set default value automatically

```go

parser, e := structarg.NewArgumentParser(&Options{},
                                        "programname",
                                        `description text`,
                                        `epilog of the program`)

e = parser.ParseArgs(os.Args[1:], false)
if e != nil {
    panic(e)
}

options := parser.Options().(*Options)

// then access argument values via options
// ...
```

# use ParseArgs2 which set default value according to parameter
# call SetDefault() to set options default value

```go

parser, e := structarg.NewArgumentParser(&Options{},
                                        "programname",
                                        `description text`,
                                        `epilog of the program`)

// do not set default value after parse
e = parser.ParseArgs2(os.Args[1:], false, false)
if e != nil {
    panic(e)
}

options := parser.Options().(*Options)

if len(options.Config) > 0 {
    e = parser.ParseFile(options.Config)
    if e != nil {
        panic(e)
    }
}

parser.SetDefault() // set option default here

// then access argument values via options
// ...

```
