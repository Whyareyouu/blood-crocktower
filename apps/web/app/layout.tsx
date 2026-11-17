import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Кровь на часовой башне",
  description: "Веб-версия настольной игры. Подготовка проекта.",
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return <html lang="ru"><body>{children}</body></html>;
}
