var values = new Array<int>(3);
values[0] = 20;
values[1] = 22;
var sum = 0;
foreach (var value in values) {
    sum += value;
}
Console.WriteLine(sum, values.Count);
