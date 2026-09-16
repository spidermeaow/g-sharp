var messages = chan<string>(1);
select {
    case messages <- "ready":
        Console.WriteLine("sent");
    default:
        Console.WriteLine("not ready");
}
select {
    case message = <-messages:
        Console.WriteLine(message);
}
