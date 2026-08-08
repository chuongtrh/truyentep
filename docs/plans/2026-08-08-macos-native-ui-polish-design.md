# Thiết kế polish giao diện theo phong cách macOS native

## Mục tiêu

Làm Web UI của Truyền Tệp có cảm giác như một tiện ích macOS chính chủ: yên tĩnh, rõ ràng, dễ thao tác và hoàn thiện ở cả light mode lẫn dark mode. Giữ nguyên kiến trúc HTML/CSS/JavaScript nhúng, API và luồng gửi dữ liệu; không thêm framework, font hoặc tài nguyên mạng ngoài.

## Hướng thị giác

Giao diện dùng nền xám trung tính, panel sáng với viền hairline và shadow mềm. Header được thu gọn theo kiểu toolbar, icon và trạng thái hoạt động trở thành các capsule nhỏ. Typography, khoảng cách, corner radius, focus ring và button state dùng một hệ thống token thống nhất. Dark mode dùng graphite thay vì đen tuyệt đối; reduced motion tiếp tục được tôn trọng.

## Thành phần và tương tác

Luồng hai bước vẫn được giữ lại. Khu vực chọn máy là card chính với peer card giống AirDrop: avatar tròn, trạng thái online, địa chỉ phụ và checkmark rõ ràng. Khu vực gửi trình bày clipboard và tệp như hai action ngang hàng; khi chưa chọn peer, action vẫn hiển thị cùng lời giải thích ngắn thay vì chỉ mờ đi. Drag/drop, keyboard navigation và progress upload tiếp tục hoạt động.

Lịch sử dùng SVG icon gửi, nhận, clipboard và lỗi thay cho ký tự văn bản. Form nhập IP, toast, empty state và dialog dùng chung visual language. JavaScript chỉ thay đổi khi cần bổ sung semantic state hoặc icon; không thay đổi endpoint hay payload.

## Kiểm chứng

Kiểm tra syntax JavaScript, Go test/vet, build gói macOS, responsive ở chiều rộng nhỏ, dark mode, focus-visible và reduced-motion. Soát lại trạng thái: đang tìm peer, không có peer, một/nhiều peer, peer được chọn, đang gửi, gửi thành công, gửi lỗi và dialog thoát.
