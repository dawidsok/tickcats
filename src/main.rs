fn main() {
    if let Err(error) = tickcats::cli::run(std::env::args_os().skip(1)) {
        eprintln!("Error: {error}");
        std::process::exit(1);
    }
}
