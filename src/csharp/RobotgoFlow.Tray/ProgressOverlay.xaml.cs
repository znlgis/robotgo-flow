using System.Runtime.InteropServices;
using System.Windows;
using System.Windows.Interop;
using RobotgoFlow.Tray.ViewModels;

namespace RobotgoFlow.Tray;

public partial class ProgressOverlay : Window
{
    private const int WS_EX_TRANSPARENT = 0x20;
    private const int WS_EX_TOOLWINDOW = 0x80;
    private const int GWL_EXSTYLE = -20;

    private readonly ProgressViewModel _vm = new();

    public ProgressOverlay()
    {
        InitializeComponent();
        DataContext = _vm;

        Loaded += OnLoaded;
    }

    private void OnLoaded(object sender, RoutedEventArgs e)
    {
        MakeTransparent();
        PositionAtBottomRight();
    }

    private void MakeTransparent()
    {
        var hwnd = new WindowInteropHelper(this).Handle;
        var extendedStyle = GetWindowLongPtr(hwnd, GWL_EXSTYLE);
        SetWindowLongPtr(hwnd, GWL_EXSTYLE, (IntPtr)((long)extendedStyle | WS_EX_TRANSPARENT | WS_EX_TOOLWINDOW));
    }

    private void PositionAtBottomRight()
    {
        var area = SystemParameters.WorkArea;
        Left = area.Right - Width - 20;
        Top = area.Bottom - Height - 20;
    }

    public ProgressViewModel ViewModel => _vm;

    // P/Invoke 声明：根据 IntPtr.Size 区分 32 位和 64 位 API
    [DllImport("user32.dll", EntryPoint = "GetWindowLongPtr", SetLastError = true)]
    private static extern IntPtr GetWindowLongPtr64(IntPtr hwnd, int index);

    [DllImport("user32.dll", EntryPoint = "GetWindowLong", SetLastError = true)]
    private static extern int GetWindowLong32(IntPtr hwnd, int index);

    [DllImport("user32.dll", EntryPoint = "SetWindowLongPtr", SetLastError = true)]
    private static extern IntPtr SetWindowLongPtr64(IntPtr hwnd, int index, IntPtr newStyle);

    [DllImport("user32.dll", EntryPoint = "SetWindowLong", SetLastError = true)]
    private static extern int SetWindowLong32(IntPtr hwnd, int index, int newStyle);

    private static IntPtr GetWindowLongPtr(IntPtr hwnd, int index)
        => IntPtr.Size == 8 ? GetWindowLongPtr64(hwnd, index) : (IntPtr)GetWindowLong32(hwnd, index);

    private static IntPtr SetWindowLongPtr(IntPtr hwnd, int index, IntPtr newStyle)
        => IntPtr.Size == 8
            ? SetWindowLongPtr64(hwnd, index, newStyle)
            : (IntPtr)SetWindowLong32(hwnd, index, (int)newStyle);
}
