# Hướng dẫn sử dụng Truyền Tệp

## Cài đặt

1. Giải nén bản phát hành và kéo **Truyền Tệp.app** vào **Applications** trên cả hai máy Mac.
2. Lần đầu mở, bấm chuột phải ứng dụng → **Open/Mở** → xác nhận **Open/Mở** vì bản tự biên dịch chỉ được ký cục bộ.
3. Cho phép truy cập **Local Network/Mạng nội bộ** khi macOS hỏi.
4. Biểu tượng hai chiều xuất hiện trên thanh trạng thái. Ứng dụng tiếp tục nhận tệp khi cửa sổ trình duyệt đã đóng.

## Gửi

1. Hai máy kết nối cùng Wi‑Fi và cùng mở Truyền Tệp.
2. Chọn máy nhận trong danh sách tự động.
3. Bấm **Gửi clipboard**, hoặc kéo thả một hay nhiều tệp vào vùng gửi.
4. Máy nhận lưu tệp trong `~/Downloads/Truyền Tệp`; clipboard nhận sẽ thay nội dung đang sao chép.

Mỗi máy có một vân tay khóa. Mọi lượt gửi đều dùng khóa phiên mới và được mã hóa/xác thực. Nếu dữ liệu bị sửa hoặc truyền thiếu, ứng dụng từ chối và xóa tệp tạm.

## Thanh trạng thái

- **Mở Truyền Tệp**: mở giao diện điều khiển.
- **Mở thư mục nhận**: mở nơi lưu tệp.
- **Thoát Truyền Tệp**: dừng nhận và gỡ biểu tượng khỏi thanh trạng thái.

## Khi hai máy không thấy nhau

- Tắt **Client/AP isolation** hoặc mạng khách trên bộ phát Wi‑Fi.
- Cho phép Truyền Tệp trong Firewall của macOS.
- Mở mục thêm bằng địa chỉ IP, nhập dạng `192.168.1.20:8777`.
- VPN có thể đổi tuyến mạng; hãy tạm ngắt VPN để thử lại.
