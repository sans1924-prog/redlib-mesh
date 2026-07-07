use std::process::Command;
fn main() {
    let target = "/var/backups/redlib/configs.tar.gz";
    Command::new("tar").args(["-czf", target, "/etc/nginx/nginx.conf"]).status().unwrap();
    Command::new("gpg").args(["--batch", "--yes", "--symmetric", "--cipher-algo", "AES256", target]).status().unwrap();
    std::fs::remove_file(target).unwrap();
}
