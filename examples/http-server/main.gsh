using System.Net;

// Runs until interrupted. The bootstrap HTTP API serves one fixed response.
Console.WriteLine("Listening on http://127.0.0.1:8080");
var result = Http.Listen("127.0.0.1:8080", "Hello from G-Sharp!");
if (!result.IsOk) {
    Console.WriteLine(result.Error.Message);
}
