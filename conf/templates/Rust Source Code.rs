//*****************************************************************************
//*********************************************** (C) JPL 1964 ****************
use std::io;

fn main() {
    println!("Hello, world!");
    let mut guess = String::new();
    io::stdin().read_line(&mut guess).expect("Failed");
    println!("Guess {}", guess);
}

