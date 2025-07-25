import Logo from "@/components/Logo.tsx";

const Header = () => {

  return (
    <header className="fixed top-0 left-0 right-0 z-50 bg-black/90 backdrop-blur-md border-b border-gray-800">
      <div className="container mx-auto px-4 py-4">
        <div className="flex items-center justify-between">
          <Logo/>
        </div>
      </div>
    </header>
  );
};

export default Header;
