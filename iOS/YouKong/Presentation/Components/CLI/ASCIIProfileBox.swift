import SwiftUI

/// ASCII 个人资料框组件
struct ASCIIProfileBox: View {
    let name: String
    let phone: String

    var body: some View {
        VStack(spacing: 0) {
            // 顶部边框
            HStack(spacing: 0) {
                Text(CLIConstants.boxTopLeft)
                Text(String(repeating: CLIConstants.boxHorizontal, count: boxWidth))
                Text(CLIConstants.boxTopRight)
            }
            .font(.system(size: 14, design: .monospaced))
            .foregroundColor(.gray)

            // 空行
            HStack(spacing: 0) {
                Text(CLIConstants.boxVertical)
                Spacer()
                    .frame(width: CGFloat(boxWidth * 7))
                Text(CLIConstants.boxVertical)
            }
            .font(.system(size: 14, design: .monospaced))
            .foregroundColor(.gray)

            // 名字
            HStack(spacing: 0) {
                Text(CLIConstants.boxVertical)
                Text("  \(name)  ")
                    .font(.system(size: 18, design: .monospaced))
                    .fontWeight(.semibold)
                    .frame(width: CGFloat(boxWidth * 7))
                Text(CLIConstants.boxVertical)
            }
            .foregroundColor(.gray)

            // 空行
            HStack(spacing: 0) {
                Text(CLIConstants.boxVertical)
                Spacer()
                    .frame(width: CGFloat(boxWidth * 7))
                Text(CLIConstants.boxVertical)
            }
            .font(.system(size: 14, design: .monospaced))
            .foregroundColor(.gray)

            // 底部边框
            HStack(spacing: 0) {
                Text(CLIConstants.boxBottomLeft)
                Text(String(repeating: CLIConstants.boxHorizontal, count: boxWidth))
                Text(CLIConstants.boxBottomRight)
            }
            .font(.system(size: 14, design: .monospaced))
            .foregroundColor(.gray)

            // 手机号
            Text(phone)
                .font(.system(size: 14, design: .monospaced))
                .foregroundColor(.secondary)
                .padding(.top, 8)
        }
    }

    private var boxWidth: Int {
        // 根据名字长度计算框宽度（中文字符算2个宽度）
        let nameWidth = name.reduce(0) { count, char in
            let scalar = char.unicodeScalars.first?.value ?? 0
            // 简单判断：大于 0x4E00 认为是中文
            return count + (scalar > 0x4E00 ? 2 : 1)
        }
        return max(nameWidth + 4, 12)
    }
}

#Preview {
    VStack(spacing: 30) {
        ASCIIProfileBox(name: "张三", phone: "13800138000")
        ASCIIProfileBox(name: "Alice", phone: "+1 234 567 8900")
    }
    .padding()
}
