import { twMerge } from 'tailwind-merge'

type LogoProps = {
  className?: string
}

const Logo = ({ className }: LogoProps) => {
  return (
    <div className={twMerge("flex items-center space-x-2", className)}>
      <div className="w-8 h-8 bg-gradient-to-r from-purple-500 to-pink-500 rounded-lg flex items-center justify-center">
        <span className="text-white font-bold text-sm">S</span>
      </div>
      <span className="text-white font-bold text-xl">Subdivx</span>
    </div>
  );
};

export default Logo;