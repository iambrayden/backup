import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./src/pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/components/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      colors: {
        brand: {
          50: "#f0f5ff",
          100: "#e0ebff",
          500: "#3b6ef5",
          600: "#2952d3",
          700: "#1e3ea8",
          900: "#111f5c",
        },
      },
    },
  },
  plugins: [],
};
export default config;
