Result<int, Error> Divide(int numerator, int denominator) {
    if (denominator == 0) {
        return Result.Err(new Error("division by zero"));
    }
    return Result.Ok(numerator / denominator);
}
Result<int, Error> Calculate() {
    var result = Divide(84, 2)?;
    return Result.Ok(result);
}
var answer = Calculate();
if (answer.IsOk) {
    Console.WriteLine(answer.Value);
} else {
    Console.WriteLine(answer.Error.Message);
}
