import type { Metadata } from "next";
import MemberView from "./view";

// Link previews normally come from the API's /m/{code} page, which carries the
// member's number; these are the fallbacks when this page is shared directly.
export const metadata: Metadata = {
  metadataBase: new URL("https://app.ayni-ai.com"),
  title: "Ayni founding member",
  description: "One of the first 1,000 people whose devices power Ayni, the open-source network for private AI on idle phones and laptops.",
  openGraph: {
    title: "Ayni founding member",
    description: "One of the first 1,000 people whose devices power Ayni.",
    images: [{ url: "/founding-og.png", width: 1200, height: 630 }],
  },
  twitter: { card: "summary_large_image" },
};

export default function MemberPage() {
  return <MemberView />;
}
