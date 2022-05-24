# Hermod YAML v1

Hermod uses YAML files to define Units and Endpoints, which are automatically compiled into your target language.

> Note: 'Field' refers to a Hermod Field object. 'field' refers to a YAML field.

## File names
Files can be nested within directories, as long as the root directory is provided to the compiler. The files in the parent category passed to the compiler, as well as files in any sub-directory, are known as the **compilation context**.

File names must end with `.hermod.yaml`. All other `.yaml` files will be ignored by the compiler.

## Packages
The first line of a Hermod YAML file must be in the following format:

```yaml
package: <package-name>
```

`package-name` refers to the general name you want to apply to the Units and Services defined within the file. **This may have implications in certain languages**; for example, in Go, this name refers to the package the output files belong to.

It's recommended to use the same package name for all files within the same directory to avoid compatibility issues.

## Imports
...are not currently supported. However, your YAML file can reference any Unit that's defined in the same compilation context.

## Naming rules
The names of primitives cannot be used in any `name` field (regardless of case).

Names can contain letters [a-z][A-Z] and numbers [0-9]. They cannot contain any other characters. Names must start with a capital letter and use camel-case throughout.

## Units
To define a unit, add an entry to the top-level `units` list:

```yaml
package: example
units:
  - name: User
    id: 0
    fields:
      - ...
```

**No two Units within the same compilation context may have the same name.** Unit names are used in compiled code, but aren't ever referred to in Hermod-encoded binary Units.

The Unit ID is used to uniquely identify a Unit when decoding a binary Unit. It's used in Endpoints to ensure the correct argument type is being delivered, as well as when matching relationships. The ID must be unique across all Units in your compilation context.

The highest supported ID number is 65535.

### Fields
Fields are defined within the `fields` list of a Unit.

```yaml
...
    fields:
      - name: displayName
        id: 0
        type: string

```

Similarly to Units, Fields also have an ID and a name. However, these don't have to be unique across the compilation context, only the Unit they're contained in. Again, the highest supported Field ID is 65535.

#### Types
All Fields must have a type. Hermod provides a number of built-in primitives to leverage cross-platform support.

| Type name (case-insensitive) | Max                  | Min                  |
|------------------------------|----------------------|----------------------|
| string                       |                      |                      |
| boolean                      | 0xff                 | 0x00                 |
| tinyinteger                  | 255                  | 0                    |
| smallinteger                 | 65535                | 0                    |
| integer                      | 4294967296           | 0                    |
| biginteger                   | 18446744073709552000 | 0                    |
| tinysignedinteger            | 127                  | -127                 |
| smallsignedinteger           | 32767                | -32767               |
| signedinteger                | 2147483648           | -2147483648          |
| bigsignedinteger             | 9223372036854776000  | -9223372036854776000 |

While most programming languages call unsigned integers "unsigned integers", Hermod swaps the naming conventions to make unsigned numbers the 'default'. Databases in production applications store signed numbers much less often, and unsigned integers are considerably more efficient for storing data.

The `type` field can also refer to the name of another Unit within the same compilation context.

#### Repeated Fields
Similarly to Protobuf, Fields can be repeated to form an array-style structure. All data types (even references to other Units) support repetition.

```yaml
...
    fields:
      - name: pronouns
        id: 1
        type: string
        repeated: true
```

#### Extended Fields
By default, Fields are encoded with a 32-bit header to specify their length in bytes. This allows a field to be up to 4294967296 bytes in length. When this is insufficient, you can increase the limit to 18446744073709552000 bytes (2^64 using a 64-bit header instead of a 32-bit header) by adding the `extended` field:

```yaml
...
    fields:
      - name: essay
        id: 3
        type: string
        extended: true
```

Extended Fields can still be repeated. However, keep in mind that this adds a significant size overhead, which is very inefficient if your Fields aren't regularly going over the 2^32 length limit.

### Go-specific features
Hermod adds some extra utilities you can use to make your Go development even more streamlined. When compiling for any language, these will just get ignored.

#### Tags
Go uses struct tags for things like JSON marshalling or some ORM packages. You can make Hermod automatically add these to your Go struct by adding a field:
```yaml
...
    fields:
      - name: password
        id: 5
        type: string
        tag: json:"password,omitempty"
```

#### Embedded structs (and imports)
You can make Hermod embed another struct inside the one being created for your Unit. This is useful for [GORM](https://gorm.io/docs/models.html#embedded_struct) for example. You'll usually also need to import a package to make this work, so you can use the `import` keyword. Imports are de-duplicated at compile time.

You can specify multiple embeds and imports and Hermod will embed/import all of them for you.

Of course, embedded field values don't actually get encoded by Hermod — they're just for your convenience.

```yaml
units:
  - name: User
    id: 0
    import: 
      - gorm.io/gorm
    embed:
      - gorm.Model
    fields:
      ...
```

You can also use an `import` at the package level:
```yaml
package: example
import: 
  - gorm.io/gorm
units:
  ...
```

## Services

At the moment, a `Service` in itself has no significance beyond grouping multiple endpoints under a common name. However, we're adding the construct in for forwards-compatibility, in case it becomes helpful to group Endpoints like this.

Services are defined at the root level, next to your Unit definitions:

```yaml
package: example
units:
  - ...
services:
  - name: MovieMetadata
    endpoints:
      - path: /movie/get-all
        id: 0
        in:
          unit: MovieSearchQuery
        out:
          unit: Movie
          streamed: true
```

### Endpoints
Endpoints are defined using a `path` field, which must be unique across your compilation context for each Endpoint. Paths must be in URL path format, similarly to the examples shown on this page.

Similarly, endpoints also contain an `id` field that must be unique across the compilation context.

Keep in mind that, when generating code, the Hermod compiler will reverse your path name: `/profile/get` will become `GetProfileRequest{}` (or something similar).

They take an `in` argument and an `out` argument. `unit` refers to the name of the Unit that the argument must be. Either, neither, or both arguments may be streamed.

Both arguments are optional. All of these constructs are valid:

```yaml
...
    endpoints:
      - path: /movie/get-latest
        id: 0
        out:
          unit: Movie
      - path: /analytics/pageview
        id: 1
      - path: /movie/add-multiple-comments
        id: 2
        in:
          unit: CommentData
          streamed: true
      - path: /chat/live
        id: 3
        in:
          unit: Message
          streamed: true
        out:
          unit: Message
          streamed: true
```

#### Streaming

Since all Hermod connections are over a WebSocket (or, in the future, over HTTP3 WebTransports), any form of bi-directional communication is supported. Unlike gRPC, this also applies to browser clients.

Of course, the handler code for each of the four possible streaming combinations will be different. The Hermod compiler will automatically generate the simplest API applicable to your choice.
