using System.Windows;
using RobotgoFlow.Tray.ViewModels;

namespace RobotgoFlow.Tray;

public partial class MiniPanelWindow : Window
{
    public MiniPanelWindow(MiniPanelViewModel vm)
    {
        InitializeComponent();
        DataContext = vm;

        vm.OnStartRequested += (path, _, _, _) =>
        {
            if (!string.IsNullOrEmpty(path))
                Hide();
        };
    }

    public void ShowNearTray()
    {
        var area = SystemParameters.WorkArea;
        Left = area.Right - Width - 10;
        Top = area.Bottom - Height - 50;
        Show();
        Activate();
    }
}
