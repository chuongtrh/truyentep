import AppKit

final class AppDelegate: NSObject, NSApplicationDelegate {
    private let statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
    private var core: Process?
    private let address = URL(string: "http://127.0.0.1:8777")!

    func applicationDidFinishLaunching(_ notification: Notification) {
        statusItem.button?.image = NSImage(systemSymbolName: "arrow.left.arrow.right.circle.fill", accessibilityDescription: "Truyền Tệp")
        statusItem.button?.image?.isTemplate = true
        let menu = NSMenu()
        menu.addItem(withTitle: "Mở Truyền Tệp", action: #selector(openApp), keyEquivalent: "o")
        menu.addItem(withTitle: "Mở thư mục nhận", action: #selector(openDownloads), keyEquivalent: "d")
        menu.addItem(.separator())
        menu.addItem(withTitle: "Đang nhận tệp trong mạng nội bộ", action: nil, keyEquivalent: "")
        menu.addItem(.separator())
        menu.addItem(withTitle: "Thoát Truyền Tệp", action: #selector(quit), keyEquivalent: "q")
        menu.items.forEach { $0.target = self }
        statusItem.menu = menu
        startCore()
    }

    private func startCore() {
        guard let path = Bundle.main.path(forResource: "truyentep-core", ofType: nil) else {
            showError("Không tìm thấy lõi ứng dụng.")
            return
        }
        let process = Process()
        process.executableURL = URL(fileURLWithPath: path)
        process.arguments = ["--no-open"]
        process.terminationHandler = { [weak self] task in
            if task.terminationStatus != 0 { DispatchQueue.main.async { self?.showError("Lõi Truyền Tệp đã dừng bất thường.") } }
        }
        do {
            try process.run(); core = process
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.55) { self.openApp() }
        } catch { showError("Không khởi động được lõi ứng dụng: \(error.localizedDescription)") }
    }

    @objc private func openApp() { NSWorkspace.shared.open(address) }

    @objc private func openDownloads() {
        let home = FileManager.default.homeDirectoryForCurrentUser
        let folder = home.appendingPathComponent("Downloads/Truyền Tệp", isDirectory: true)
        try? FileManager.default.createDirectory(at: folder, withIntermediateDirectories: true)
        NSWorkspace.shared.open(folder)
    }

    @objc private func quit() {
        var request = URLRequest(url: address.appendingPathComponent("api/quit"))
        request.httpMethod = "POST"
        request.setValue("1", forHTTPHeaderField: "X-Truyen-Tep-Local")
        URLSession.shared.dataTask(with: request).resume()
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.25) {
            if self.core?.isRunning == true { self.core?.terminate() }
            NSApplication.shared.terminate(nil)
        }
    }

    private func showError(_ message: String) {
        let alert = NSAlert(); alert.messageText = "Truyền Tệp"; alert.informativeText = message; alert.runModal()
    }
}

let app = NSApplication.shared
let delegate = AppDelegate()
app.delegate = delegate
app.setActivationPolicy(.accessory)
app.run()
