# Thiết kế cấu hình version ứng dụng

## Mục tiêu

Tạo một nguồn version duy nhất cho Truyền Tệp. Build macOS phải đọc version từ file cấu hình thay vì hard-code hoặc nhận version ứng dụng từ biến môi trường. Web UI phải hiển thị đúng version đã được nhúng trong binary đang chạy.

## Cấu hình

File `version.json` ở thư mục gốc chứa hai trường:

```json
{
  "version": "0.3.0",
  "build": 3
}
```

`version` là semantic version dùng cho tên artifact, Go core và `CFBundleShortVersionString`. `build` là số nguyên dương dùng cho `CFBundleVersion`. Build dừng sớm với thông báo rõ nếu file thiếu, JSON sai, version không hợp lệ hoặc build number không hợp lệ.

## Luồng build

`scripts/build-macos.sh` đọc `version.json` bằng công cụ có sẵn trên macOS, kiểm tra dữ liệu rồi truyền version vào Go bằng `-ldflags`. Script cập nhật cả hai khóa trong app bundle và đặt tên ZIP theo version. Test đóng gói có thể thêm suffix riêng vào tên thư mục/artifact để tránh đè bản phát hành, nhưng version bên trong app luôn đến từ cấu hình.

## Hiển thị UI

Go core tiếp tục trả `Version` qua `/api/state`. HTML thêm vị trí version ở footer; JavaScript render `Phiên bản <version>` từ state thực tế, không đọc trực tiếp file JSON. Cách này đảm bảo UI phản ánh binary đang chạy kể cả khi file nguồn đã thay đổi sau build.

## Kiểm chứng

Test khóa cấu trúc `version.json`, semantic hook và JavaScript render version. Build test xác nhận script đọc cả version/build, tên artifact đúng, Info.plist trong app đúng và config không hợp lệ bị từ chối. README hướng dẫn cập nhật duy nhất `version.json` trước khi phát hành.
