use std::net::{SocketAddr, TcpStream};
use std::process::{Child, Command};
use std::sync::Mutex;
use std::time::Duration;
use tauri::path::BaseDirectory;
use tauri::{AppHandle, Manager, RunEvent};

const SERVER_ADDR: &str = "127.0.0.1:8787";
const CREATE_NO_WINDOW: u32 = 0x0800_0000;

struct ServerProcess(Mutex<Option<Child>>);

fn port_is_busy() -> bool {
    let addr: SocketAddr = SERVER_ADDR.parse().unwrap();
    TcpStream::connect_timeout(&addr, Duration::from_millis(300)).is_ok()
}

fn start_server(app: &AppHandle) -> Option<Child> {
    if port_is_busy() {
        return None;
    }
    let exe = app
        .path()
        .resolve("resources/mirai-server.exe", BaseDirectory::Resource)
        .ok()?;
    if !exe.exists() {
        return None;
    }
    let data_dir = app.path().data_dir().ok()?.join("Mirai");
    std::fs::create_dir_all(&data_dir).ok()?;
    let mut command = Command::new(exe);
    command.arg("-addr").arg(SERVER_ADDR).arg("-data").arg(&data_dir);
    #[cfg(windows)]
    {
        use std::os::windows::process::CommandExt;
        command.creation_flags(CREATE_NO_WINDOW);
    }
    command.spawn().ok()
}

fn stop_server(app: &AppHandle) {
    if let Some(state) = app.try_state::<ServerProcess>() {
        if let Ok(mut guard) = state.0.lock() {
            if let Some(mut child) = guard.take() {
                let _ = child.kill();
                let _ = child.wait();
            }
        }
    }
}

pub fn run() {
    let app = tauri::Builder::default()
        .setup(|app| {
            let child = start_server(app.handle());
            app.manage(ServerProcess(Mutex::new(child)));
            Ok(())
        })
        .build(tauri::generate_context!())
        .expect("не удалось запустить Mirai");

    app.run(|handle, event| {
        if let RunEvent::Exit = event {
            stop_server(handle);
        }
    });
}
