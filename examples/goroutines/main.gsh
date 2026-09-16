void Work(chan<int> output, int value) {
    output <- value * 2;
}
var result = chan<int>();
go Work(result, 21);
Console.WriteLine(<-result);
