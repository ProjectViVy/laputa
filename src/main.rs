use clap::Parser;
use laputa::cli::Cli;

fn main() {
    let cli = Cli::parse();
    match cli.run() {
        Ok(output) => {
            if !output.is_empty() {
                println!("{output}");
            }
        }
        Err(error) => {
            eprintln!("{error}");
            std::process::exit(1);
        }
    }
}
