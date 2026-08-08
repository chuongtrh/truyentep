# Thiết kế script publish GitHub Release

## Mục tiêu

Thêm `scripts/publish-release.sh` để phát hành phiên bản macOS được khai báo trong `version.json`. Script build ứng dụng bằng quy trình hiện có, tạo tag Git và GitHub Release, rồi đính kèm file ZIP vừa build. Nếu tag tương ứng đã tồn tại trên remote `origin`, script phải dừng trước khi build hoặc thay đổi Git/GitHub.

## Luồng xử lý

Script chạy từ bất kỳ thư mục hiện hành nào nhưng luôn thao tác trong repository chứa chính nó. Phiên bản được đọc qua `scripts/read-version.sh`; tên tag là `v<version>`. Preflight kiểm tra dependency, working tree sạch, remote `origin`, tag trên remote, trạng thái đồng bộ của commit hiện tại với upstream và GitHub CLI đã đăng nhập. Kiểm tra tag dùng `git ls-remote --exit-code --tags origin refs/tags/<tag>` để phát hiện phiên bản do máy khác đã publish, thay vì chỉ nhìn tag local.

Sau khi preflight thành công, script chạy `go test ./...` và `scripts/test-version-config.sh`, rồi gọi `scripts/build-macos.sh`. Đường dẫn ZIP được lấy từ dòng output cuối của build script và được xác nhận là một file nằm trong `dist/`. Script tạo annotated tag tại commit hiện tại, push tag lên `origin`, sau đó gọi `gh release create <tag> <zip> --verify-tag --generate-notes --title <tag>`.

Nếu tag đã có, thông báo lỗi nêu rõ version và remote rồi thoát khác 0. Nếu build/test thất bại, không tạo tag. Nếu push tag thất bại, không tạo release. Nếu `gh release create` thất bại sau khi tag đã được push, script giữ tag remote để không thực hiện thao tác phá hủy và in lệnh retry với artifact hiện tại.

## Kiểm thử

Kiểm thử shell chạy script trong repository Git tạm, đưa các command bên ngoài vào `PATH` bằng executable giả lập và ghi lại lời gọi. Các ca chính gồm: từ chối tag đã tồn tại trước khi build; từ chối working tree bẩn; dừng khi test/build lỗi; và luồng thành công gọi build, tạo/push annotated tag, rồi tạo release kèm đúng ZIP. Ngoài test script chuyên biệt, toàn bộ Go test và test version config hiện có vẫn phải chạy xanh.
