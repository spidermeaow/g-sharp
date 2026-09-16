var messages = chan<string>(1);
messages <- "hello through a channel";
Console.WriteLine(<-messages);
