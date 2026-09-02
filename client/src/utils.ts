import { theme } from "antd";

const shanghaiDateTimeFormatter = new Intl.DateTimeFormat("en-CA", {
  timeZone: "Asia/Shanghai",
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit",
  second: "2-digit",
  hourCycle: "h23",
});

export const getMonacoThemeMode = (color: string): "vs" | "vs-dark" => {
  let hex = color.replace("#", "");
  if (hex.length === 3) {
    hex = hex
      .split("")
      .map((part) => part + part)
      .join("");
  }

  const red = parseInt(hex.substring(0, 2), 16);
  const green = parseInt(hex.substring(2, 4), 16);
  const blue = parseInt(hex.substring(4, 6), 16);

  if (isNaN(red) || isNaN(green) || isNaN(blue)) return "vs";

  const brightness = (red * 299 + green * 587 + blue * 114) / 1000;

  return brightness < 128 ? "vs-dark" : "vs";
};

export const useMonacoTheme = () => {
  const { token } = theme.useToken();
  return getMonacoThemeMode(token.colorBgBase);
};

export const formatDate24h = (dateStr: string): string => {
  const date = new Date(dateStr);
  const parts = Object.fromEntries(
    shanghaiDateTimeFormatter
      .formatToParts(date)
      .map(({ type, value }) => [type, value]),
  );
  return `${parts.year}-${parts.month}-${parts.day} ${parts.hour}:${parts.minute}:${parts.second}`;
};

export const formatBytes = (bytes: number, decimals = 2): string => {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ["B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + " " + sizes[i];
};
