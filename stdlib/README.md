# Bootstrap standard library facade

Signatures are resolved by the G-Sharp checker. The Go backend emits runtime
implementations; programs cannot name arbitrary Go packages.

| Namespace | API |
| --- | --- |
| System | Console.WriteLine(values...), Console.Write(values...), Error.Message |
| System.IO | File.ReadAllText(string path) → Result<string,Error> |
| System.Collections | Array<T>, List<T>.Add(T), Map<K,V>, Count and indexing |
| System.Threading | chan<T>, go and select language primitives |
| System.Time | Time.Sleep(int milliseconds) |
| System.Net | Http.Listen(string address, string response) → Result<int,Error> |

For bootstrap usability, these facade names are predeclared; `using System...;`
is permitted but not required. Console supports primitive and enum values. Count
on strings counts UTF-8 bytes in v0.1. Http.Listen blocks while serving, uses a
private mux, and returns a typed error if startup/serving fails. This minimal API
is for examples; a general HTTP request/response abstraction is future work.
