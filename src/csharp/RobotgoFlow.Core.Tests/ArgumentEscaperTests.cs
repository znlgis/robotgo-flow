using RobotgoFlow.Core.Helpers;
using Xunit;

namespace RobotgoFlow.Core.Tests;

public class ArgumentEscaperTests
{
    [Theory]
    [InlineData("", "\"\"")]
    [InlineData(null, "\"\"")]
    public void Quote_空字符串和null_返回空引号对(string? input, string expected)
    {
        var result = ArgumentEscaper.Quote(input!);
        Assert.Equal(expected, result);
    }

    [Theory]
    [InlineData("hello", "\"hello\"")]
    [InlineData("simple", "\"simple\"")]
    [InlineData("C:\\Program Files\\app.exe", "\"C:\\Program Files\\app.exe\"")]
    public void Quote_普通无空格路径_直接包裹引号(string input, string expected)
    {
        var result = ArgumentEscaper.Quote(input);
        Assert.Equal(expected, result);
    }

    [Fact]
    public void Quote_包含空格_正确包裹引号()
    {
        var result = ArgumentEscaper.Quote("hello world");
        Assert.Equal("\"hello world\"", result);
    }

    [Fact]
    public void Quote_包含双引号_正确转义()
    {
        var result = ArgumentEscaper.Quote("he\"llo");
        Assert.Equal("\"he\\\"llo\"", result);
    }

    [Fact]
    public void Quote_已有包裹引号的路径_正确转义内部引号()
    {
        var result = ArgumentEscaper.Quote("\"already quoted\"");
        Assert.Equal("\"\\\"already quoted\\\"\"", result);
    }

    [Fact]
    public void Quote_仅双引号_正确包裹()
    {
        var result = ArgumentEscaper.Quote("\"");
        Assert.Equal("\"\\\"\"", result);
    }

    [Fact]
    public void Quote_末尾反斜杠_正确翻倍转义()
    {
        // 末尾反斜杠需要翻倍，防止转义闭合引号
        var result = ArgumentEscaper.Quote("C:\\Program Files\\");
        Assert.Equal("\"C:\\Program Files\\\\\"", result);
    }

    [Fact]
    public void Quote_反斜杠后接双引号_正确转义()
    {
        // 反斜杠后紧跟双引号：反斜杠翻倍再加一个转义反斜杠
        var result = ArgumentEscaper.Quote("path\\\"");
        Assert.Equal("\"path\\\\\\\"\"", result);
    }

    [Fact]
    public void Quote_连续反斜杠在中间_保持原样()
    {
        // 普通前导反斜杠（非末尾、非引号前）保持原样
        var result = ArgumentEscaper.Quote("C:\\\\Data\\file.txt");
        Assert.Equal("\"C:\\\\Data\\file.txt\"", result);
    }

    [Fact]
    public void Quote_多个反斜杠在末尾_全部翻倍()
    {
        var result = ArgumentEscaper.Quote("trail\\\\");
        Assert.Equal("\"trail\\\\\\\\\"", result);
    }

    [Fact]
    public void Quote_中文文件名_正确包裹()
    {
        var result = ArgumentEscaper.Quote("我的文档\\文件.txt");
        Assert.Equal("\"我的文档\\文件.txt\"", result);
    }

    [Fact]
    public void Quote_特殊字符_正确包裹()
    {
        var result = ArgumentEscaper.Quote("test&pause");
        Assert.Equal("\"test&pause\"", result);
    }

    [Fact]
    public void Quote_仅反斜杠_正确翻倍()
    {
        // 单个反斜杠在末尾，翻倍为两个
        var result = ArgumentEscaper.Quote("\\");
        Assert.Equal("\"\\\\\"", result);
    }

    [Fact]
    public void Quote_控制字符_Tab和换行_原始保留()
    {
        var result = ArgumentEscaper.Quote("a\tb\nc");
        Assert.Equal("\"a\tb\nc\"", result);
    }
}
