// Built-in generic types are supported. User-defined generics are deferred.
var values = new List<int>();
values.Add(20);
values.Add(22);
var total = 0;
foreach (var value in values) {
    total += value;
}
var scores = new Map<string, int>();
scores["answer"] = total;
Console.WriteLine(scores["answer"]);
