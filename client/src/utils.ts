import { theme } from "antd";

export const useMonacoTheme = () => {
  const { token } = theme.useToken();

  const getThemeMode = (color: string) => {
    // 1. 去掉 # 号并处理 3 位简写 (如 #000 -> 000000)
    let hex = color.replace("#", "");
    if (hex.length === 3) {
      hex = hex
        .split("")
        .map((s) => s + s)
        .join("");
    }

    // 2. 提取 RGB 值
    const r = parseInt(hex.substring(0, 2), 16);
    const g = parseInt(hex.substring(2, 4), 16);
    const b = parseInt(hex.substring(4, 6), 16);

    // 3. 如果解析失败（比如不是有效的 Hex），默认返回亮色
    if (isNaN(r) || isNaN(g) || isNaN(b)) return "vs";

    // 4. YIQ 亮度算法：计算结果在 0-255 之间
    const brightness = (r * 299 + g * 587 + b * 114) / 1000;

    return brightness < 128 ? "vs-dark" : "vs";
  };

  const monacoTheme = getThemeMode(token.colorBgBase);

  console.log(
    "Current Background:",
    token.colorBgBase,
    "Monaco Theme:",
    monacoTheme,
  );

  return monacoTheme;
};

export const formatDate24h = (dateStr: string): string => {
  const date = new Date(dateStr);
  const pad = (num: number) => num.toString().padStart(2, "0");
  const y = date.getFullYear();
  const m = pad(date.getMonth() + 1);
  const d = pad(date.getDate());
  const h = pad(date.getHours());
  const min = pad(date.getMinutes());
  const s = pad(date.getSeconds());
  return `${y}-${m}-${d} ${h}:${min}:${s}`;
};

export const formatBytes = (bytes: number, decimals = 2): string => {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ["B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + " " + sizes[i];
};
