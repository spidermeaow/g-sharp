namespace Demo;

interface Processor {
    Result<int, Error> Process(int value);
}

class Worker {
    Result<int, Error> Process(int value) {
        return Result.Ok(value * 2);
    }
}

Result<int, Error> ProcessJob(Processor worker, int value) {
    var answer = worker.Process(value)?;
    return Result.Ok(answer);
}

class Program {
    static void Main() {
        var jobs = chan<int>(10);
        var results = chan<Result<int, Error>>(10);
        go {
            var value = <-jobs;
            Processor worker = new Worker();
            results <- ProcessJob(worker, value);
        };
        jobs <- 21;
        var answer = <-results;
        if (answer.IsOk) {
            Console.WriteLine(answer.Value);
        } else {
            Console.WriteLine(answer.Error.Message);
        }
    }
}
