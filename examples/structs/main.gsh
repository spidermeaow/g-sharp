struct Point {
    int X;
    int Y;
    int Sum() {
        return X + Y;
    }
}
var point = new Point();
point.X = 20;
point.Y = 22;
var copy = point;
copy.X = 100;
Console.WriteLine(point.Sum(), copy.Sum());
