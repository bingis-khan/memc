use fmt;
use os;

export fn main() void = {
	const argnum = len(os::args);
	if (argnum < 2) {
		fail("Must specify a command.");
	};

	const action = os::args[1];
	switch (action) {
		case "i", "init" => initaction();
		case "a", "annotate" => {
			fmt::println("annotate!!")!;
		};
		case "s", "status" => {
			fmt::println("status!!")!;
		};
		case "g", "ignore" => {
			fmt::println("ignore!!")!;
		};
		case "f", "find" => {
			fmt::println("find!!")!;
		};
		case => {
			fail("Unknown command");
		};
	};
};

fn initaction() void = {
	fmt::println("init action!!")!;
};


fn fail(reason: str) void = {
	fmt::println(reason)!;
	os::exit(1);
};
