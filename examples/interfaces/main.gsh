interface Processor {
    int Process(int value);
}
class Worker {
    int Process(int value) {
        return value * 2;
    }
}
Processor processor = new Worker();
Console.WriteLine(processor.Process(21));
