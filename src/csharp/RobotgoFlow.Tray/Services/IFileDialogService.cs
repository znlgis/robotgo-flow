using Microsoft.Win32;

namespace RobotgoFlow.Tray.Services;

public interface IFileDialogService
{
    string? ShowOpenFileDialog(string filter);
}

public class FileDialogService : IFileDialogService
{
    public string? ShowOpenFileDialog(string filter)
    {
        var dlg = new Microsoft.Win32.OpenFileDialog
        {
            Title = "选择工作流文件",
            Filter = filter
        };
        return dlg.ShowDialog() == true ? dlg.FileName : null;
    }
}
