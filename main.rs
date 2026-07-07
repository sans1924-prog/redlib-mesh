use std::process::Command;

fn main() {
    // Task 2 Fix: Ensures the directory layout exists so tar doesn't fail
    if let Err(e) = std::fs::create_dir_all("/var/backups/redlib") {
        eprintln!("Warning: Failed to create backup directory: {}", e);
    }

    let target = "/var/backups/redlib/configs.tar.gz";
    Command::new("tar").args(["-czf", target, "/etc/nginx/nginx.conf"]).status().unwrap();
    Command::new("gpg").args(["--batch", "--yes", "--symmetric", "--cipher-algo", "AES256", target]).status().unwrap();
    std::fs::remove_file(target).unwrap();
}
