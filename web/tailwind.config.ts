import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        bg: "#0b0f1a",
        surface: "#131928",
        border: "#1e293b",
        accent: "#3b82f6",
        "accent-hover": "#2563eb",
      },
    },
  },
  plugins: [],
};

export default config;
