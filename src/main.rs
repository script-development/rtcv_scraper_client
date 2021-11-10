use std::io;

fn main() {
    println!(r#"{{"type":"ready","msg":"waiting for credentials"}}"#);
    let mut buffer = String::new();
    let stdin = io::stdin();
    loop {
        buffer.clear();
        stdin.read_line(&mut buffer).unwrap();
        buffer = buffer.trim().to_string();
        println!("{}", buffer);
    }
}
