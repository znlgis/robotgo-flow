using System.Collections.Generic;
using System.Text.Json.Serialization;

namespace RobotgoFlow.Core.Models;

public record ServeEvent(
    [property: JsonPropertyName("type")] string Type,
    [property: JsonPropertyName("ok")] bool? Ok,
    [property: JsonPropertyName("name")] string? Name,
    [property: JsonPropertyName("total_steps")]
    int TotalSteps,
    [property: JsonPropertyName("inputs")] List<InputInfo>? Inputs,
    [property: JsonPropertyName("idx")] int Idx,
    [property: JsonPropertyName("step_idx")]
    int StepIdx,
    [property: JsonPropertyName("total")] int Total,
    [property: JsonPropertyName("action")] string? Action,
    [property: JsonPropertyName("detail")] string? Detail,
    [property: JsonPropertyName("level")] string? Level,
    [property: JsonPropertyName("message")]
    string? Message,
    [property: JsonPropertyName("error")] string? Error,
    [property: JsonPropertyName("estimated_sec")]
    double EstimatedSec,
    [property: JsonPropertyName("screenshot_path")]
    string? ScreenshotPath,
    [property: JsonPropertyName("total_elapsed_sec")]
    double TotalElapsedSec
);

public record InputInfo(
    [property: JsonPropertyName("name")] string Name,
    [property: JsonPropertyName("label")] string Label,
    [property: JsonPropertyName("placeholder")]
    string? Placeholder,
    [property: JsonPropertyName("required")]
    bool Required,
    [property: JsonPropertyName("mask")] bool Mask
);

public record ServeCommand(
    [property: JsonPropertyName("type")] string Type,
    [property: JsonPropertyName("values")] Dictionary<string, string>? Values
);


