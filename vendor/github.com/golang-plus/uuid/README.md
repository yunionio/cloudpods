# uuid

Package **uuid** implements UUID [RFC 4122](http://www.ietf.org/rfc/rfc4122.txt).

## Usage

### Generating

#### Time-Based (Version 1)

    uuid.NewTimeBased() (uuid.UUID, error)
    uuid.NewV1() (uuid.UUID, error)

#### DCE Security (Version 2)

    uuid.NewDCESecurity(uuid.Domain) (uuid.UUID, error)
    uuid.NewV2(uuid.Domain) (uuid.UUID, error)

#### Name-Based uses MD5 hashing (Version 3)

    uuid.NewNameBasedMD5(namespace, name string) (uuid.UUID, error)
    uuid.NewV3(namespace, name string) (uuid.UUID, error)

#### Random (Version 4)

    uuid.NewRandom() (uuid.UUID, error)
    uuid.NewV4() (uuid.UUID, error)

#### Name-Based uses SHA-1 hashing (Version 5)

    uuid.NewNameBasedSHA1(namespace, name string) (uuid.UUID, error)
    uuid.NewV5(namespace, name string) (uuid.UUID, error)

### Styles

* Standard: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (8-4-4-4-12, length: 36)
* Without Dash: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx (length: 32)

### Formatting & Parsing

    uuid.UUID.String() string                        // format to standard style
    uuid.UUID.Format(uuid.Style) string              // format to uuid.StyleStandard or uuid.StyleWithoutDash

    uuid.Parse(string) (uuid.UUID, error)       // parse from UUID string

