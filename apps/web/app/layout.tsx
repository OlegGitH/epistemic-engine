import type { Metadata } from "next";
import "@xyflow/react/dist/style.css";
import "./styles.css";

export const metadata: Metadata = {
  title: "Epistemic Control Center",
  description: "Account-level AI knowledge, assurance, and certification.",
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return <html lang="en"><body>{children}</body></html>;
}
