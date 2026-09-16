class Counter {
    int Value { get; private set; }
    void Increment() {
        Value++;
    }
}
var counter = new Counter();
var shared = counter;
shared.Increment();
Console.WriteLine(counter.Value);
