import filecmp

def compare_dirs(dir1, dir2):
    # 创建一个dircmp对象来比较两个目录
    comparison = filecmp.dircmp(dir1, dir2)

    # 检查是否有不匹配的文件
    if comparison.diff_files or comparison.left_only or comparison.right_only:
        print("Directories are NOT identical.")
        # 打印不匹配的文件和独有的文件
        print("Different files:", comparison.diff_files)
        print("Files only in", dir1, ":", comparison.left_only)
        print("Files only in", dir2, ":", comparison.right_only)
    else:
        print("Directories are identical.")

# 比较两个目录
compare_dirs('D:\PostgraduateWORK\Go\PBFT - ordinary\PBFT\\network', 'D:\PostgraduateWORK\Go\PBFT\PBFT\\network')
