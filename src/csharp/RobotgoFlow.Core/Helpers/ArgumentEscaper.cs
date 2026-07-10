using System.Text;

namespace RobotgoFlow.Core.Helpers;

/// <summary>
///     命令行参数转义工具，防止参数注入。
///     测试覆盖：空/null、空格、双引号、反斜杠、连续反斜杠、Unicode、控制字符。
/// </summary>
public static class ArgumentEscaper
{
    /// <summary>
    ///     将参数用双引号包裹，严格遵循 Windows CommandLineToArgvW 转义规则。
    ///     正确处理反斜杠在前导双引号前、以及末尾反斜杠的情况。
    /// </summary>
    public static string Quote(string arg)
    {
        if (string.IsNullOrEmpty(arg)) return "\"\"";

        var sb = new StringBuilder();
        sb.Append('"');
        for (var i = 0; i < arg.Length; i++)
        {
            var c = arg[i];
            if (c == '\\')
            {
                // 统计连续反斜杠数量
                var backslashCount = 1;
                while (i + 1 < arg.Length && arg[i + 1] == '\\')
                {
                    backslashCount++;
                    i++;
                }

                // 反斜杠在字符串末尾：翻倍防止转义闭合引号
                if (i + 1 == arg.Length)
                {
                    sb.Append('\\', backslashCount * 2);
                }
                // 反斜杠后紧跟双引号：翻倍 + 1（一个转义引号，其余转义反斜杠自身）
                else if (arg[i + 1] == '"')
                {
                    sb.Append('\\', backslashCount * 2 + 1);
                    sb.Append('"');
                    i++; // 跳过双引号
                }
                // 普通前导反斜杠：保持原样
                else
                {
                    sb.Append('\\', backslashCount);
                }
            }
            else if (c == '"')
            {
                // 不在反斜杠前缀后的独立双引号：用反斜杠转义
                sb.Append("\\\"");
            }
            else
            {
                sb.Append(c);
            }
        }

        sb.Append('"');
        return sb.ToString();
    }
}
