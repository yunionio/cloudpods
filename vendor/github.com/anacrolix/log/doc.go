/*
Package log implements a std log compatible logging system that draws some inspiration from the
[Python logging module] from Python's standard library. It supports multiple handlers, log levels,
zero-allocation, scopes, custom formatting, and environment and runtime configuration.

When not used to replace std log, the import should use the package name "analog" as in:

	import analog "github.com/anacrolix/log"

# Names

Each Logger has a sequence of names that are used for filtering and context. Names are commonly attached as Loggers are passed into code of deeper context. The full import path of the package where a message is generated, and the short source file name and line number are added as the last 2 names for each message (applying any [Msg.Skip] in finding the right frame) when filtering is applied. The names are included at the end of each logging line.

# Rules

A sequence of rules are parsed from the environment variable with the key of [EnvRules]. Rules are separated by ",". Each rule is a substring of a log message name that or "*" to match any name. If there is no "=" in the rule, then all messages that match will be logged. If there is a "=", then a message must have the level following the "=", as parsed by [Level.UnmarshalText] or higher to be logged. Each rule is checked in order, and the last match takes precedence. This helps when you want to chain new rules on existing ones, you can always append to the end to override earlier rules.

	GO_LOG := "" | rule ("," rule)*
	rule := filter ("=" level) | ""
	filter := name | "*"
	level := "all", "debug" | "info" | "warn" | "err" | "crit" | see [Level.UnmarshalText]

Some examples:

  - GO_LOG=*

    Log everything, no matter the level

  - GO_LOG=*=,*,hello=debug

    Log everything, except for any message containing a name with the substring "hello", which must be at least debug level.

  - GO_LOG=something=info

    Handle messages at the info level or greater if they have a name containing "something".

If no rule matches, the [Logger]'s filter level is checked. The [Default] filter level is [Warning]. This means only messages with the level of [Warning] or higher will be logged, unless overridden by the specific Logger in use, or a rule from the environment matches.

# Rule reporting

If the environment variable with the key [EnvReportRules] is not the empty string, each message logged with a previously unseen permutation of names will output a message to a standard library logger with the minimum level required to log that permutation. The message itself is then handled as usual. The same permutation will not be reported on again. This is useful to determine what logging names are in use, and to debug their reporting level thresholds.

[Python logging module]: https://docs.python.org/3/library/logging.html
*/
package log
